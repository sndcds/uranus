package api

import (
	"bufio"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func TestOrgPartnerRequestEmailTemplates(t *testing.T) {
	t.Chdir("..")
	for _, templateName := range []string{"partner-request", "partner-request-accepted"} {
		for _, locale := range []string{"de", "en", "da"} {
			t.Run(templateName+"/"+locale, func(t *testing.T) {
				data := orgPartnerRequestEmailData{
					Language: locale, DisplayName: "Alex <script>", FromOrgName: "From & Co",
					ToOrgName: "To <b>Org</b>", Message: "Hello\n<script>alert(1)</script>",
					RedirectLink: "https://dashboard.example.test/admin/org/partners",
				}
				subject, body, err := app.RenderEmailTemplate(
					"template/email/layout/"+locale+".html", "template/email/"+templateName+"/"+locale+".html", data,
				)
				if err != nil {
					t.Fatal(err)
				}
				for _, expected := range []string{`lang="` + locale + `"`, "Alex &lt;script&gt;", "From &amp; Co", "To &lt;b&gt;Org&lt;/b&gt;", data.RedirectLink, `class="legal"`} {
					if !strings.Contains(body, expected) {
						t.Errorf("email is missing %q", expected)
					}
				}
				if subject == "" || strings.Contains(body, "<script>") {
					t.Fatal("empty subject or unescaped user input")
				}
				if templateName == "partner-request" && !strings.Contains(body, "Hello\n&lt;script&gt;alert(1)&lt;/script&gt;") {
					t.Fatal("request message missing or not escaped")
				}
				data.Message = ""
				_, body, err = app.RenderEmailTemplate("template/email/layout/"+locale+".html", "template/email/"+templateName+"/"+locale+".html", data)
				if err != nil || strings.Contains(body, "white-space:pre-wrap") {
					t.Fatalf("empty message must be omitted: %v", err)
				}
			})
		}
	}
}

type partnerTestEmail struct {
	To, Subject, Body string
}

// Accept SMTP locally so integration tests exercise the real asynchronous
// delivery path without sending email to an external service.
func partnerTestSMTP(t *testing.T, config *app.Config) <-chan partnerTestEmail {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	config.AuthSmtpHost = "127.0.0.1"
	config.AuthSmtpPort, _ = strconv.Atoi(port)
	config.AuthSmtpLogin = "sender@example.test"
	config.AuthReplyEmail = config.AuthSmtpLogin
	emails := make(chan partnerTestEmail, 32)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				reader := textproto.NewReader(bufio.NewReader(conn))
				fmt.Fprint(conn, "220 localhost ESMTP\r\n")
				for {
					line, err := reader.ReadLine()
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"):
						fmt.Fprint(conn, "250-localhost\r\n250 AUTH PLAIN\r\n")
					case strings.HasPrefix(line, "AUTH"):
						fmt.Fprint(conn, "235 authenticated\r\n")
					case strings.Contains(line, "reject@example.test"):
						fmt.Fprint(conn, "550 rejected for test\r\n")
					case line == "DATA":
						fmt.Fprint(conn, "354 send message\r\n")
						data, err := reader.ReadDotBytes()
						if err != nil {
							return
						}
						message, err := mail.ReadMessage(strings.NewReader(string(data)))
						if err != nil {
							return
						}
						body, _ := io.ReadAll(message.Body)
						subject, _ := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
						emails <- partnerTestEmail{To: message.Header.Get("To"), Subject: subject, Body: string(body)}
						fmt.Fprint(conn, "250 accepted\r\n")
					case line == "QUIT":
						fmt.Fprint(conn, "221 goodbye\r\n")
						return
					default:
						fmt.Fprint(conn, "250 OK\r\n")
					}
				}
			}()
		}
	}()
	return emails
}

func partnerRequestDatabase(t *testing.T) (*ApiHandler, *gin.Engine, <-chan partnerTestEmail) {
	t.Helper()
	h, _ := authDatabase(t)
	dbExec(t, h, "CREATE EXTENSION IF NOT EXISTS postgis")
	for _, table := range []string{"organization", "user_organization_link", "organization_member_link", "organization_partner_request", "organization_access_grants"} {
		ddl, err := os.ReadFile("../ddl/" + table + ".ddl")
		if err != nil {
			t.Fatal(err)
		}
		dbExec(t, h, strings.Split(string(ddl), "-- Indices")[0])
	}
	query, err := os.ReadFile("../sql/admin-insert-org-partner-request.sql")
	if err != nil {
		t.Fatal(err)
	}
	app.UranusInstance.SqlAdminInsertOrgPartnerRequest = strings.ReplaceAll(string(query), "{{schema}}", h.DbSchema)
	for _, org := range []struct{ id, name string }{{socialOrg, "Requesting Org"}, {socialOtherOrg, "Requested Org"}} {
		dbExec(t, h, "INSERT INTO uranus.organization (uuid,name) VALUES ($1,$2)", org.id, org.name)
	}
	h.Config.FrontendDashboard = "https://dashboard.example.test/"
	emails := partnerTestSMTP(t, h.Config)
	app.UranusInstance.Config = *h.Config
	r := gin.New()
	r.Use(func(gc *gin.Context) { gc.Set("user-uuid", authTestUser) })
	r.POST("/org/:orgUuid/request", h.AdminInsertOrgPartnerRequest)
	r.POST("/org/:orgUuid/accept/:partnerUuid", h.AdminInsertOrgPartnerAccept)
	t.Chdir("..")
	return h, r, emails
}

func receivePartnerEmail(t *testing.T, emails <-chan partnerTestEmail) partnerTestEmail {
	t.Helper()
	select {
	case email := <-emails:
		return email
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for partner email")
		return partnerTestEmail{}
	}
}

func noPartnerEmail(t *testing.T, emails <-chan partnerTestEmail) {
	t.Helper()
	select {
	case email := <-emails:
		t.Fatalf("unexpected email: %+v", email)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestOrgPartnerRequestPostgresNotifications(t *testing.T) {
	h, r, emails := partnerRequestDatabase(t)
	en, da, de, fr := "en", "da", "de", "fr"
	for i, fixture := range []struct {
		locale     *string
		org        string
		permission app.Permissions
		joined     bool
	}{
		{&en, socialOtherOrg, app.UserPermAnswerPartnerRequest, true},
		{&da, socialOtherOrg, app.UserPermCombinationAdmin, true},
		{nil, socialOtherOrg, app.UserPermAnswerPartnerRequest, true},
		{&fr, socialOtherOrg, app.UserPermAnswerPartnerRequest, true},
		{&de, socialOtherOrg, app.UserPermRequestPartner, true},
		{&de, socialOtherOrg, app.UserPermAnswerPartnerRequest, false},
		{&de, socialOrg, app.UserPermAnswerPartnerRequest, true},
	} {
		id := fmt.Sprintf("01994126-6680-7000-8000-%012d", i+10)
		dbExec(t, h, `INSERT INTO uranus."user" (uuid,email,password_hash,locale,display_name)
			VALUES ($1,$2,'unused',$3,'Recipient')`, id, fmt.Sprintf("recipient%d@example.test", i), fixture.locale)
		dbExec(t, h, "INSERT INTO uranus.user_organization_link (user_uuid,org_uuid,permissions) VALUES ($1,$2,$3)", id, fixture.org, int64(fixture.permission))
		dbExec(t, h, "INSERT INTO uranus.organization_member_link (user_uuid,org_uuid,has_joined) VALUES ($1,$2,$3)", id, fixture.org, fixture.joined)
	}
	// Duplicate permission rows must not produce duplicate emails.
	dbExec(t, h, `INSERT INTO uranus.user_organization_link SELECT * FROM uranus.user_organization_link WHERE user_uuid='01994126-6680-7000-8000-000000000010'`)
	body := `{"to_org_uuid":"` + socialOtherOrg + `","message":"Hello <script>\nPartner"}`
	assertSocialStatus(t, socialRequest(r, "POST", "/org/"+socialOrg+"/request", "", body), 201)
	seen := map[string]bool{}
	for range 4 {
		email := receivePartnerEmail(t, emails)
		locale := map[string]string{"recipient0@example.test": "en", "recipient1@example.test": "da", "recipient2@example.test": "de", "recipient3@example.test": "de"}[email.To]
		if locale == "" || seen[email.To] || !strings.Contains(email.Body, `lang="`+locale+`"`) ||
			!strings.Contains(email.Body, "Hello &lt;script&gt;\nPartner") ||
			!strings.Contains(email.Body, "https://dashboard.example.test/admin/org/partners") {
			t.Fatalf("wrong recipient, locale or content: %+v", email)
		}
		seen[email.To] = true
	}
	noPartnerEmail(t, emails)
	assertSocialStatus(t, socialRequest(r, "POST", "/org/"+socialOrg+"/request", "", body), 409)
	noPartnerEmail(t, emails)

	// Acceptance must notify the stored requester, even if someone else accepts.
	dbExec(t, h, `UPDATE uranus."user" SET locale='en',first_name='Original',last_name='Requester' WHERE uuid=$1`, authTestUser)
	r.Use(func(gc *gin.Context) { gc.Set("user-uuid", socialOtherUser) })
	r.POST("/other/:orgUuid/accept/:partnerUuid", h.AdminInsertOrgPartnerAccept)
	path := "/other/" + socialOtherOrg + "/accept/" + socialOrg
	assertSocialStatus(t, socialRequest(r, "POST", path, "", ""), 200)
	email := receivePartnerEmail(t, emails)
	if email.To != "auth@example.test" || email.Subject != "kulturbytes – Your partnership request has been accepted" ||
		!strings.Contains(email.Body, "Hello Original Requester,") || !strings.Contains(email.Body, "Requested Org") || !strings.Contains(email.Body, "Requesting Org") {
		t.Fatalf("wrong acceptance notification: %+v", email)
	}
	assertSocialStatus(t, socialRequest(r, "POST", path, "", ""), 409)
	noPartnerEmail(t, emails)
}

func TestOrgPartnerRequestPostgresRollbackNotifications(t *testing.T) {
	h, r, emails := partnerRequestDatabase(t)
	dbExec(t, h, `CREATE FUNCTION uranus.fail_partner_commit() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'test commit failure'; END $$;
		CREATE CONSTRAINT TRIGGER fail_partner_commit AFTER INSERT OR UPDATE ON uranus.organization_partner_request
		DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION uranus.fail_partner_commit()`)
	body := `{"to_org_uuid":"` + socialOtherOrg + `"}`
	assertSocialStatus(t, socialRequest(r, "POST", "/org/"+socialOrg+"/request", "", body), 500)
	noPartnerEmail(t, emails)
	dbExec(t, h, "DROP TRIGGER fail_partner_commit ON uranus.organization_partner_request")
	dbExec(t, h, `INSERT INTO uranus.organization_partner_request (from_user_uuid,from_org_uuid,to_org_uuid) VALUES ($1,$2,$3)`, authTestUser, socialOrg, socialOtherOrg)
	dbExec(t, h, `CREATE CONSTRAINT TRIGGER fail_partner_commit AFTER UPDATE ON uranus.organization_partner_request
		DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION uranus.fail_partner_commit()`)
	assertSocialStatus(t, socialRequest(r, "POST", "/org/"+socialOtherOrg+"/accept/"+socialOrg, "", ""), 500)
	noPartnerEmail(t, emails)
}

func TestOrgPartnerRequestPostgresDeliveryFailure(t *testing.T) {
	h, _, emails := partnerRequestDatabase(t)
	dbExec(t, h, `INSERT INTO uranus.organization_partner_request (from_user_uuid,from_org_uuid,to_org_uuid) VALUES ($1,$2,$3)`, authTestUser, socialOrg, socialOtherOrg)
	for i, email := range []string{"reject@example.test", "success@example.test"} {
		id := fmt.Sprintf("01994126-6680-7000-8000-%012d", i+10)
		dbExec(t, h, `INSERT INTO uranus."user" (uuid,email,password_hash) VALUES ($1,$2,'unused')`, id, email)
		dbExec(t, h, "INSERT INTO uranus.user_organization_link (user_uuid,org_uuid,permissions) VALUES ($1,$2,$3)", id, socialOtherOrg, int64(app.UserPermAnswerPartnerRequest))
		dbExec(t, h, "INSERT INTO uranus.organization_member_link (user_uuid,org_uuid,has_joined) VALUES ($1,$2,true)", id, socialOtherOrg)
	}
	if err := h.sendOrgPartnerRequestEmails(socialOrg, socialOtherOrg); err == nil {
		t.Fatal("expected delivery error")
	}
	if email := receivePartnerEmail(t, emails); email.To != "success@example.test" {
		t.Fatalf("other recipient was not notified: %+v", email)
	}
}
