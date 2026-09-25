package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func TestSocialPreviewAuthenticationAndUUID(t *testing.T) {
	h := setupAuthTest(t)
	r := socialPostRouter(h)
	path := socialPostsPath + "/" + authTestUser + "/preview"
	for _, token := range []string{"", "invalid"} {
		assertSocialStatus(t, socialRequest(r, "POST", path, token, ""), 401)
	}
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/invalid/preview", socialToken(t, authTestUser), ""), 400)
	req := httptest.NewRequest("POST", path, nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: socialToken(t, authTestUser)})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assertSocialStatus(t, w, 403)
}
func previewPost(t *testing.T, r *gin.Engine, token string, accounts ...string) model.SocialPost {
	t.Helper()
	body := strings.Replace(socialPostBody(socialOrg, "event", accounts...), authTestUser, contentEvent, 1)
	w := socialRequest(r, "POST", socialPostsPath, token, body)
	assertSocialStatus(t, w, 201)
	return socialPostData(t, w)
}
func previewData(t *testing.T, w *httptest.ResponseRecorder) model.SocialPostPreview {
	t.Helper()
	assertSocialStatus(t, w, 200)
	var response struct {
		Data         model.SocialPostPreview `json:"data"`
		ResponseType string                  `json:"response_type"`
		Status       int                     `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.ResponseType != "admin-preview-social-post" || response.Status != 200 || response.Data.Previews == nil {
		t.Fatal(w.Body.String())
	}
	return response.Data
}

type previewRejectNetwork struct{ t *testing.T }

func (r previewRejectNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	r.t.Error("preview attempted an external HTTP request")
	return nil, fmt.Errorf("network forbidden in preview test")
}

func TestSocialPreviewPostgresReadOnly(t *testing.T) {
	h, r, token := contentDatabase(t)
	captureSocialLogs(t)
	app.UranusInstance.Config.FrontendClient = "https://events.example.test"
	accounts := []string{}
	for _, platform := range []string{"facebook", "instagram", "mastodon", "bluesky"} {
		id := createPostAccount(t, r, token, platform, socialOrg)
		accounts = append(accounts, id)
		if platform == "mastodon" {
			dbExec(t, h, "UPDATE uranus.social_account SET base_url='https://social.example.test' WHERE uuid=$1", id)
		}
	}
	// Distinct accounts on the same platform each need their own preview.
	w := socialRequest(r, "POST", socialAccountsPath, token, socialBody("facebook", socialOrg, "second-facebook"))
	assertSocialStatus(t, w, 201)
	tokenless := socialAccountData(t, w).Uuid
	accounts = append(accounts, tokenless)
	dbExec(t, h, "UPDATE uranus.social_account SET access_token=NULL,refresh_token=NULL WHERE uuid=$1", tokenless)
	// Keep this all-platform fixture below the strict Bluesky budget.
	dbExec(t, h, "UPDATE uranus.event SET ticket_link=NULL WHERE uuid=$1", contentEvent)
	dbExec(t, h, "UPDATE uranus.event_date SET ticket_link=NULL WHERE event_uuid=$1", contentEvent)
	post := previewPost(t, r, token, accounts...)
	dbExec(t, h, `UPDATE uranus.social_post_target SET status='published', published_at='2026-09-24T12:00:00Z',
  scheduled_at='2026-09-23T12:00:00Z',remote_post_id='old-remote-id',error='old error' WHERE social_post_uuid=$1`, post.Uuid)
	path := socialPostsPath + "/" + post.Uuid
	before := socialPostData(t, socialRequest(r, "GET", path, token, ""))
	// Make writes fail at the database, including writes that preserve row values.
	dbExec(t, h, `CREATE FUNCTION uranus.reject_preview_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'preview wrote data'; END $$`)
	for _, table := range []string{"social_post", "social_post_target", "social_account"} {
		dbExec(t, h, "CREATE TRIGGER forbid_preview_write BEFORE INSERT OR UPDATE OR DELETE OR TRUNCATE ON uranus."+table+" FOR EACH STATEMENT EXECUTE FUNCTION uranus.reject_preview_write()")
	}
	originalTransport := http.DefaultTransport
	http.DefaultTransport = previewRejectNetwork{t}
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	first := previewData(t, socialRequest(r, "POST", path+"/preview", token, ""))
	second := previewData(t, socialRequest(r, "POST", path+"/preview", token, ""))
	if first.PostUuid != post.Uuid || len(first.Previews) != len(accounts) || !reflect.DeepEqual(first, second) {
		t.Fatalf("unexpected or non-deterministic preview: %+v", first)
	}
	for i, p := range first.Previews {
		if p.TargetUuid != post.Targets[i].Uuid || p.Platform == "" || p.Text == "" || p.ImageURL == "" || !strings.Contains(p.URL, "/de/veranstaltung/") {
			t.Fatalf("unexpected target: %+v", p)
		}
	}
	after := socialPostData(t, socialRequest(r, "GET", path, token, ""))
	if !reflect.DeepEqual(before, after) {
		t.Fatal("preview changed post or targets")
	}
	for _, account := range accounts {
		if account == tokenless {
			storedSocialTokens(t, h, account, "", "")
		} else {
			storedSocialTokens(t, h, account, "secret-access", "secret-refresh")
		}
	}
	// A target-specific failure also leaves every post and target untouched.
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	w = socialRequest(r, "POST", path+"/preview", token, "")
	assertSocialStatus(t, w, 400)
	var instagramTarget string
	for _, target := range post.Targets {
		if target.SocialAccountUuid == accounts[1] {
			instagramTarget = target.Uuid
		}
	}
	if instagramTarget == "" || !strings.Contains(w.Body.String(), instagramTarget) || !strings.Contains(w.Body.String(), "instagram") || !strings.Contains(w.Body.String(), "image is required") || strings.Contains(w.Body.String(), "previews") {
		t.Fatal(w.Body.String())
	}
	if !reflect.DeepEqual(before, socialPostData(t, socialRequest(r, "GET", path, token, ""))) {
		t.Fatal("failed preview changed post")
	}
}

func TestSocialPreviewPostgresLocalizationAndSources(t *testing.T) {
	h, r, token := contentDatabase(t)
	app.UranusInstance.Config.FrontendClient = "https://events.example.test"
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid
	dbExec(t, h, "UPDATE uranus.event SET content_iso_639_1='da',title='Jazzaften',ticket_link='https://tickets.test' WHERE uuid=$1", contentEvent)
	for _, tc := range []struct{ query, route, label string }{{"", "da/begivenhed", "Billetter"}, {"?lang=en", "en/event", "Tickets"}, {"?lang=unknown", "de/veranstaltung", "Tickets"}} {
		got := previewData(t, socialRequest(r, "POST", path+"/preview"+tc.query, token, "")).Previews[0]
		if !strings.Contains(got.URL, tc.route) || !strings.Contains(got.Text, tc.label) || !strings.Contains(got.Text, "Jazzaften") {
			t.Fatalf("unexpected localization: %+v", got)
		}
	}
	for _, source := range []struct{ kind, id string }{{"venue", contentVenue}, {"organization", socialOrg}} {
		assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"source_type":"`+source.kind+`","source_uuid":"`+source.id+`"}`), 200)
		if got := previewData(t, socialRequest(r, "POST", path+"/preview", token, "")).Previews[0]; got.Text == "" || got.URL == "" {
			t.Fatal(got)
		}
	}
}

func TestSocialPreviewPostgresFailures(t *testing.T) {
	h, r, token := contentDatabase(t)
	captureSocialLogs(t)
	account := createPostAccount(t, r, token, "mastodon", socialOrg)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid
	assertFailure := func(status int, reason string) {
		t.Helper()
		w := socialRequest(r, "POST", path+"/preview", token, "")
		assertSocialStatus(t, w, status)
		if !strings.Contains(w.Body.String(), reason) || strings.Contains(w.Body.String(), "previews") {
			t.Fatal(w.Body.String())
		}
	}
	assertFailure(400, "base_url")
	dbExec(t, h, "UPDATE uranus.social_account SET base_url='https://social.example.test',enabled=false WHERE uuid=$1", account)
	assertFailure(400, "disabled")
	dbExec(t, h, "UPDATE uranus.social_account SET enabled=true WHERE uuid=$1", account)
	dbExec(t, h, "UPDATE uranus.event SET summary=$1 WHERE uuid=$2", strings.Repeat("a", 501), contentEvent)
	assertFailure(400, "500 characters")
	assertSocialStatus(t, socialRequest(r, "POST", path+"/preview", socialToken(t, socialOtherUser), ""), 403)
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+socialOtherUser+"/preview", token, ""), 404)
	for _, source := range []string{`{"source_uuid":"` + socialOtherUser + `"}`, `{"source_type":"venue","source_uuid":"` + contentOtherVenue + `"}`} {
		assertSocialStatus(t, socialRequest(r, "PUT", path, token, source), 200)
		assertFailure(404, "source not found")
	}
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"targets":[]}`), 200)
	assertFailure(400, "no targets")
	// Missing/failed relations must not expose raw database errors or partial data.
	post = previewPost(t, r, token, account)
	path = socialPostsPath + "/" + post.Uuid
	dbExec(t, h, "ALTER TABLE uranus.event_date RENAME TO hidden_event_date")
	assertFailure(500, "internal server error")
	dbExec(t, h, "ALTER TABLE uranus.hidden_event_date RENAME TO event_date")
	gc := contentContext(authTestUser)
	ctx, cancel := context.WithCancel(gc.Request.Context())
	cancel()
	gc.Request = gc.Request.WithContext(ctx)
	gc.Params = gin.Params{{Key: "uuid", Value: post.Uuid}}
	h.AdminPreviewSocialPost(gc)
	if gc.Writer.Status() != 500 {
		t.Fatalf("cancelled preview returned %d", gc.Writer.Status())
	}
}
