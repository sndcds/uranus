package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

type onboardingFixture struct {
	h                                     *ApiHandler
	router                                *gin.Engine
	org, member, inviter, outsider, token string
}

// Uses an isolated schema, real PostgreSQL row locks and the shipped migrations.
// URANUS_TEST_DATABASE_URL must point to a disposable test database.
func onboardingDB(t *testing.T) *onboardingFixture {
	t.Helper()
	dsn := os.Getenv("URANUS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set URANUS_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "onboarding_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("CREATE SCHEMA " + schema)
	previous := app.UranusInstance
	t.Cleanup(func() {
		app.UranusInstance = previous
		_, _ = pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		pool.Close()
	})
	read := func(path string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", path))
		if err != nil {
			t.Fatal(err)
		}
		return strings.ReplaceAll(string(data), "{{schema}}", schema)
	}
	exec(fmt.Sprintf(`CREATE TABLE %s.organization (uuid uuid PRIMARY KEY, name text NOT NULL, city text, country text, web_link text, contact_email text)`, schema))
	for _, table := range []string{"user", "organization_member_link", "user_organization_link", "system_email_template"} {
		source := read("ddl/" + table + ".ddl")
		start := strings.Index(source, "CREATE TABLE")
		end := strings.Index(source[start:], ");") + start + 2
		exec(strings.ReplaceAll(source[start:end], "uranus.", schema+"."))
	}
	for _, file := range []string{"20260910_team_onboarding.sql", "20260910_team_onboarding_templates.sql"} {
		exec(strings.ReplaceAll(read("migrations/"+file), `:"schema"`, schema))
	}
	cfg := app.DefaultConfig()
	cfg.JwtSecret = "test-only-secret"
	cfg.Frontend = "https://dashboard.example"
	cfg.DbSchema = schema
	app.UranusInstance = &app.Uranus{Config: cfg, JwtKey: []byte(cfg.JwtSecret), SqlGetSystemEmailTemplate: read("sql/get-system-email-template.sql"), SqlGetUserOrgPermissions: read("sql/admin-get-user-org-permissions.sql"), SqlAdminGetOrgMemberLink: read("sql/admin-get-org-member-link.sql"), SqlAdminGetOrgMembers: read("sql/admin-get-org-members.sql"), SqlAdminInvitedOrgTeamMember: read("sql/admin-invited-org-team-member.sql"), SqlAdminUpsertInvitedOrgTeamMember: read("sql/admin-upsert-invited-org-team-member.sql")}
	f := &onboardingFixture{h: &ApiHandler{Config: &cfg, DbPool: pool, DbSchema: schema}, org: uuid.NewString(), member: uuid.NewString(), inviter: uuid.NewString(), outsider: uuid.NewString()}
	for i, id := range []string{f.inviter, f.member, f.outsider} {
		exec(fmt.Sprintf(`INSERT INTO %s."user" (uuid, email, password_hash, display_name, locale) VALUES ($1, $2, 'unused', $3, $4)`, schema), id, fmt.Sprintf("user%d@example.test", i), []string{"Inviter", "Anna <Example>", "Other"}[i], []string{"de", "da", "en"}[i])
	}
	exec(fmt.Sprintf(`INSERT INTO %s.organization (uuid, name) VALUES ($1, 'Culture & Arts')`, schema), f.org)
	f.token = f.sign(t, time.Now().Add(time.Hour))
	exec(fmt.Sprintf(`INSERT INTO %s.organization_member_link (org_uuid, user_uuid, invited_by_user_uuid, accept_token) VALUES ($1, $2, $3, $4)`, schema), f.org, f.member, f.inviter, f.token)
	exec(fmt.Sprintf(`INSERT INTO %s.organization_member_link (org_uuid, user_uuid, has_joined) VALUES ($1, $2, true)`, schema), f.org, f.inviter)
	exec(fmt.Sprintf(`INSERT INTO %s.user_organization_link (org_uuid, user_uuid, permissions) VALUES ($1, $2, $3)`, schema), f.org, f.inviter, int64(app.UserPermManageTeam|app.UserPermManagePermissions))
	gin.SetMode(gin.TestMode)
	grains_api.Init(grains_api.Config{ServiceName: "test", APIVersion: "1"})
	f.router = gin.New()
	f.router.POST("/accept", f.h.OrgTeamInviteAccept)
	group := f.router.Group("/api/admin", app.JWTMiddleware)
	group.GET("/user/notifications", f.h.AdminGetNotifications)
	group.PATCH("/user/notifications/:notificationUuid/read", f.h.AdminReadNotification)
	group.PATCH("/user/notifications/:notificationUuid/dismiss", f.h.AdminDismissNotification)
	group.GET("/org/:orgUuid/member/:memberUuid/permissions", f.h.AdminGetOrgMemberPermissions)
	group.PUT("/org/:orgUuid/member/:memberUuid/permissions", f.h.AdminUpdateOrgMemberPermissions)
	group.POST("/org/:orgUuid/team/invite", f.h.AdminOrgTeamInvite)
	group.GET("/org/:orgUuid/team", f.h.AdminGetOrgTeam)
	return f
}

func (f *onboardingFixture) sign(t *testing.T, expiry time.Time) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, OrganizationTeamInviteClaims{UserUuid: f.member, OrgUuid: f.org, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiry)}}).SignedString([]byte(f.h.Config.JwtSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func (f *onboardingFixture) call(method, path, user, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != "" {
		token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, app.Claims{UserUuid: user, TokenType: "access", RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte(f.h.Config.JwtSecret))
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	f.router.ServeHTTP(res, req)
	return res
}
func (f *onboardingFixture) accept(token string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"token": token})
	return f.call("POST", "/accept", "", string(body))
}
func (f *onboardingFixture) count(t *testing.T, table string) int {
	t.Helper()
	var count int
	if err := f.h.DbPool.QueryRow(context.Background(), "SELECT count(*) FROM "+f.h.DbSchema+"."+table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
func requireStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status %d, want %d; body %s", response.Code, want, response.Body)
	}
}

func TestTeamAcceptMembershipNotificationsAndEmail(t *testing.T) {
	f := onboardingDB(t)
	requireStatus(t, f.accept(f.token), 200)
	var joined bool
	var token *string
	var perms int64
	if err := f.h.DbPool.QueryRow(context.Background(), fmt.Sprintf(`SELECT oml.has_joined, oml.accept_token, uol.permissions FROM %s.organization_member_link oml JOIN %s.user_organization_link uol USING (org_uuid, user_uuid) WHERE oml.user_uuid = $1`, f.h.DbSchema, f.h.DbSchema), f.member).Scan(&joined, &token, &perms); err != nil {
		t.Fatal(err)
	}
	if !joined || token != nil || perms != 0 {
		t.Fatalf("incorrect membership: joined=%v token cleared=%v permissions=%d", joined, token == nil, perms)
	}
	if f.count(t, "user_notification") != 2 || f.count(t, "notification_email_outbox") != 2 || f.count(t, "user_organization_link") != 2 {
		t.Fatal("incorrect counts")
	}
	requireStatus(t, f.accept(f.token), 401)
	// Defensively exercise the database event dedupe even if processing is repeated.
	tx, err := f.h.DbPool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.h.enqueueTeamOnboarding(context.Background(), tx, f.token, f.org, "Culture & Arts", f.member, &f.inviter); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.count(t, "user_notification") != 2 || f.count(t, "notification_email_outbox") != 2 {
		t.Fatal("duplicate work")
	}
	inviter := f.call("GET", "/api/admin/user/notifications", f.inviter, "")
	requireStatus(t, inviter, 200)
	if !strings.Contains(inviter.Body.String(), "/admin/org/"+f.org+"/member/"+f.member+"/permissions") || strings.Contains(inviter.Body.String(), "@example.test") || strings.Contains(inviter.Body.String(), f.token) {
		t.Fatal("invalid inviter notification")
	}
	member := f.call("GET", "/api/admin/user/notifications", f.member, "")
	requireStatus(t, member, 200)
	if !strings.Contains(member.Body.String(), `"action_url":"/admin/orgs"`) {
		t.Fatal("invalid member action")
	}
	sent := map[string]string{}
	sender := func(to, subject, body string, _ time.Duration) error { sent[to] = subject + "\n" + body; return nil }
	if err := f.h.processNotificationEmails(context.Background(), sender); err != nil {
		t.Fatal(err)
	}
	if err := f.h.processNotificationEmails(context.Background(), sender); err != nil {
		t.Fatal(err)
	}
	if len(sent) != 2 || !strings.Contains(sent["user0@example.test"], "Berechtigungen festlegen") || !strings.Contains(sent["user1@example.test"], "Åbn organisation") || !strings.Contains(sent["user0@example.test"], "Anna &lt;Example&gt;") || strings.Contains(sent["user1@example.test"], "{{") {
		t.Fatalf("unexpected rendered templates: %v", sent)
	}
	requireStatus(t, f.call("GET", "/api/admin/org/"+f.org+"/team", f.inviter, ""), 200)
	team := f.call("GET", "/api/admin/org/"+f.org+"/team", f.inviter, "")
	if !strings.Contains(team.Body.String(), `"permissions_missing":true`) {
		t.Fatal("missing team permission status")
	}
}

func TestTeamAcceptInvalidAndExpired(t *testing.T) {
	f := onboardingDB(t)
	for _, token := range []string{"invalid", f.sign(t, time.Now().Add(-time.Hour)), f.sign(t, time.Now().Add(2*time.Hour))} {
		requireStatus(t, f.accept(token), 401)
	}
	noExpiry, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, OrganizationTeamInviteClaims{UserUuid: f.member, OrgUuid: f.org}).SignedString([]byte(f.h.Config.JwtSecret))
	requireStatus(t, f.accept(noExpiry), 401)
	if f.count(t, "user_notification") != 0 || f.count(t, "notification_email_outbox") != 0 || f.count(t, "user_organization_link") != 1 {
		t.Fatal("invalid acceptance had side effects")
	}
}

func TestTeamAcceptConcurrent(t *testing.T) {
	f := onboardingDB(t)
	var success atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if f.accept(f.token).Code == 200 {
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 || f.count(t, "user_notification") != 2 || f.count(t, "notification_email_outbox") != 2 || f.count(t, "user_organization_link") != 2 {
		t.Fatal("concurrent accept was not idempotent")
	}
	var sends atomic.Int32
	sender := func(_, _, _ string, _ time.Duration) error { sends.Add(1); return nil }
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := f.h.processNotificationEmails(context.Background(), sender); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if sends.Load() != 2 {
		t.Fatalf("sent %d mails", sends.Load())
	}
}

func TestNotificationOwnershipAndLifecycle(t *testing.T) {
	f := onboardingDB(t)
	requireStatus(t, f.accept(f.token), 200)
	var id string
	if err := f.h.DbPool.QueryRow(context.Background(), fmt.Sprintf(`SELECT uuid FROM %s.user_notification WHERE user_uuid = $1`, f.h.DbSchema), f.member).Scan(&id); err != nil {
		t.Fatal(err)
	}
	base := "/api/admin/user/notifications"
	requireStatus(t, f.call("GET", base, "", ""), 401)
	other := f.call("GET", base+"?user_uuid="+f.member, f.outsider, "")
	requireStatus(t, other, 200)
	if strings.Contains(other.Body.String(), id) {
		t.Fatal("other user saw notification")
	}
	for _, action := range []string{"read", "dismiss"} {
		requireStatus(t, f.call("PATCH", base+"/"+id+"/"+action, f.outsider, ""), 404)
	}
	requireStatus(t, f.call("PATCH", base+"/"+id+"/read", f.member, ""), 200)
	read := f.call("PATCH", base+"/"+id+"/read", f.member, "")
	requireStatus(t, read, 200)
	unread := f.call("GET", base+"?status=unread", f.member, "")
	if strings.Contains(unread.Body.String(), id) {
		t.Fatal("read still unread")
	}
	active := f.call("GET", base, f.member, "")
	if !strings.Contains(active.Body.String(), id) {
		t.Fatal("reading hid notification")
	}
	requireStatus(t, f.call("PATCH", base+"/"+id+"/dismiss", f.member, ""), 200)
	active = f.call("GET", base, f.member, "")
	if strings.Contains(active.Body.String(), id) {
		t.Fatal("dismissed reappeared")
	}
	requireStatus(t, f.call("PATCH", base+"/"+id+"/dismiss", f.member, ""), 200)
	dismissed := f.call("GET", base+"?status=dismissed", f.member, "")
	if !strings.Contains(dismissed.Body.String(), id) {
		t.Fatal("dismissed state missing")
	}
	requireStatus(t, f.call("GET", base+"?status=invalid", f.member, ""), 400)
	requireStatus(t, f.call("PATCH", base+"/invalid/read", f.member, ""), 400)
}

func TestEmailFailureDoesNotUndoMembershipOrRetry(t *testing.T) {
	f := onboardingDB(t)
	requireStatus(t, f.accept(f.token), 200)
	sends := 0
	sender := func(_, _, _ string, _ time.Duration) error { sends++; return errors.New("simulated timeout") }
	for i := 0; i < 2; i++ {
		if err := f.h.processNotificationEmails(context.Background(), sender); err != nil {
			t.Fatal(err)
		}
	}
	if sends != 2 || f.count(t, "user_organization_link") != 2 || f.count(t, "user_notification") != 2 {
		t.Fatal("failure rolled back membership or retried SMTP")
	}
	var failed int
	_ = f.h.DbPool.QueryRow(context.Background(), fmt.Sprintf(`SELECT count(*) FROM %s.notification_email_outbox WHERE status = 'failed'`, f.h.DbSchema)).Scan(&failed)
	if failed != 2 {
		t.Fatal("failed work not retained")
	}
}

func TestEmailPreparationCanResume(t *testing.T) {
	f := onboardingDB(t)
	requireStatus(t, f.accept(f.token), 200)
	f.h.Config.Frontend = ""
	sends := 0
	sender := func(_, _, _ string, _ time.Duration) error { sends++; return nil }
	if err := f.h.processNotificationEmails(context.Background(), sender); err == nil {
		t.Fatal("invalid config accepted")
	}
	if sends != 0 {
		t.Fatal("sent before preparation")
	}
	f.h.Config.Frontend = "https://dashboard.example"
	if err := f.h.processNotificationEmails(context.Background(), sender); err != nil {
		t.Fatal(err)
	}
	if sends != 2 {
		t.Fatal("pending work not resumed")
	}
}

func TestTeamPermissionEndpointsRemainProtected(t *testing.T) {
	f := onboardingDB(t)
	requireStatus(t, f.accept(f.token), 200)
	path := "/api/admin/org/" + f.org + "/member/" + f.member + "/permissions"
	requireStatus(t, f.call("GET", path, f.member, ""), 403)
	requireStatus(t, f.call("PUT", path, f.member, `{"bit":5,"enabled":true}`), 403)
	requireStatus(t, f.call("GET", path, f.inviter, ""), 200)
	requireStatus(t, f.call("PUT", path, f.inviter, `{"bit":1,"enabled":true}`), 200)
	requireStatus(t, f.call("POST", "/api/admin/org/"+f.org+"/team/invite", f.member, `{"email":"user2@example.test","referer":"https://dashboard.example"}`), 403)
}

func TestNotificationTemplateRenderingAndURL(t *testing.T) {
	subject, body := renderNotificationEmail("Hi {{display_name}}", "<p>{{display_name}}</p>", map[string]string{"display_name": "A\r\n<b>"})
	if strings.ContainsAny(subject, "\r\n") || !strings.Contains(body, "&lt;b&gt;") {
		t.Fatal("unsafe rendering")
	}
	for _, lang := range []string{"de", "en", "da"} {
		if emailLocale(lang) != lang {
			t.Fatal("locale lost")
		}
	}
	if emailLocale("fr") != "en" {
		t.Fatal("missing fallback")
	}
	h := &ApiHandler{Config: &app.Config{}}
	for _, base := range []string{"", "javascript:alert(1)", "//evil.example", "https://user:pass@example.test", "https://example.test?x=1"} {
		h.Config.Frontend = base
		if _, err := h.dashboardURL("/admin/orgs"); err == nil {
			t.Fatal("unsafe base URL")
		}
	}
	h.Config.Frontend = "https://dashboard.example/"
	if link, err := h.dashboardURL("/admin/orgs"); err != nil || link != "https://dashboard.example/admin/orgs" {
		t.Fatal("invalid URL")
	}
}

func TestInviteThenAcceptFullAPIFlow(t *testing.T) {
	f := onboardingDB(t)
	ctx := context.Background()
	_, err := f.h.DbPool.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.system_email_template (context, iso_639_1, subject, template) VALUES ('team-invite', 'en', 'Team invitation', '{{invite_link}} {{display_name}} {{organization_name}} {{expiry_minutes}}')`, f.h.DbSchema))
	if err != nil {
		t.Fatal(err)
	}
	var invitation string
	f.h.emailSender = func(_, _, body string, _ time.Duration) error { invitation = body; return nil }
	path := "/api/admin/org/" + f.org + "/team/invite"
	requireStatus(t, f.call("POST", path, f.inviter, `{"email":"user2@example.test","referer":"https://dashboard.example"}`), 201)
	if !strings.Contains(invitation, "https://dashboard.example/app/activate/team-invitation?token=") {
		t.Fatal("invite URL regressed")
	}
	var token string
	if err := f.h.DbPool.QueryRow(ctx, fmt.Sprintf(`SELECT accept_token FROM %s.organization_member_link WHERE user_uuid = $1 AND org_uuid = $2`, f.h.DbSchema), f.outsider, f.org).Scan(&token); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.accept(token), 200)
	requireStatus(t, f.accept(token), 401)
	notifications := f.call("GET", "/api/admin/user/notifications", f.inviter, "")
	if !strings.Contains(notifications.Body.String(), "/member/"+f.outsider+"/permissions") {
		t.Fatal("CTA does not match newly invited member")
	}
	requireStatus(t, f.call("PUT", "/api/admin/org/"+f.org+"/member/"+f.outsider+"/permissions", f.inviter, `{"bit":1,"enabled":true}`), 200)
}

func TestAcceptanceRollsBackIfNotificationCannotPersist(t *testing.T) {
	f := onboardingDB(t)
	// A deliberately failing constraint exercises the real transaction boundary.
	_, err := f.h.DbPool.Exec(context.Background(), fmt.Sprintf(`ALTER TABLE %s.user_notification ADD CONSTRAINT test_failure CHECK (false)`, f.h.DbSchema))
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.accept(f.token), 500)
	if f.count(t, "user_notification") != 0 || f.count(t, "notification_email_outbox") != 0 || f.count(t, "user_organization_link") != 1 {
		t.Fatal("partial acceptance persisted")
	}
	_, err = f.h.DbPool.Exec(context.Background(), fmt.Sprintf(`ALTER TABLE %s.user_notification DROP CONSTRAINT test_failure`, f.h.DbSchema))
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.accept(f.token), 200) // Token and membership were rolled back.
}

func TestExistingPermissionsAreNotOverwritten(t *testing.T) {
	f := onboardingDB(t)
	_, err := f.h.DbPool.Exec(context.Background(), fmt.Sprintf(`INSERT INTO %s.user_organization_link (user_uuid, org_uuid, permissions) VALUES ($1, $2, 2)`, f.h.DbSchema), f.member, f.org)
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.accept(f.token), 200)
	var permissions int64
	_ = f.h.DbPool.QueryRow(context.Background(), fmt.Sprintf(`SELECT permissions FROM %s.user_organization_link WHERE user_uuid = $1 AND org_uuid = $2`, f.h.DbSchema), f.member, f.org).Scan(&permissions)
	if permissions != 2 || f.count(t, "user_organization_link") != 2 {
		t.Fatal("existing permissions overwritten or duplicated")
	}
}

func TestAllEmailTemplateLocales(t *testing.T) {
	for _, lang := range []string{"de", "en", "da", "fr"} {
		t.Run(lang, func(t *testing.T) {
			f := onboardingDB(t)
			_, err := f.h.DbPool.Exec(context.Background(), fmt.Sprintf(`UPDATE %s."user" SET locale = $1`, f.h.DbSchema), lang)
			if err != nil {
				t.Fatal(err)
			}
			requireStatus(t, f.accept(f.token), 200)
			count := 0
			err = f.h.processNotificationEmails(context.Background(), func(_, subject, body string, _ time.Duration) error {
				count++
				if !strings.Contains(body, `lang="`+emailLocale(lang)+`"`) || strings.Contains(subject+body, "{{") || !strings.Contains(body, "https://dashboard.example/admin/") {
					t.Error("incorrect template/locale/variables")
				}
				return nil
			})
			if err != nil || count != 2 {
				t.Fatalf("delivery count %d, err %v", count, err)
			}
		})
	}
}

func TestClaimedEmailIsNotResentAfterRestart(t *testing.T) {
	f := onboardingDB(t)
	requireStatus(t, f.accept(f.token), 200)
	_, err := f.h.DbPool.Exec(context.Background(), fmt.Sprintf(`UPDATE %s.notification_email_outbox SET status = 'sending', attempted_at = now()`, f.h.DbSchema))
	if err != nil {
		t.Fatal(err)
	}
	err = f.h.processNotificationEmails(context.Background(), func(_, _, _ string, _ time.Duration) error { t.Error("resent ambiguous delivery"); return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeletedInviterDoesNotBlockJoining(t *testing.T) {
	f := onboardingDB(t)
	_, err := f.h.DbPool.Exec(context.Background(), fmt.Sprintf(`DELETE FROM %s."user" WHERE uuid = $1`, f.h.DbSchema), f.inviter)
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.accept(f.token), 200)
	if f.count(t, "user_notification") != 1 || f.count(t, "notification_email_outbox") != 1 {
		t.Fatal("expected welcome only for deleted inviter")
	}
}
