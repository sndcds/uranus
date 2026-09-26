package api

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func publishDatabase(t *testing.T) (*ApiHandler, *gin.Engine, string) {
	t.Helper()
	h, r, token := contentDatabase(t)
	applySocialPublishMigration(t, h, "up")
	applySocialPublicationMigration(t, h, "up")
	captureSocialLogs(t)
	return h, r, token
}

func applySocialPublishMigration(t *testing.T, h *ApiHandler, direction string) {
	t.Helper()
	sql, err := os.ReadFile("../migrations/202609250003_social_publish." + direction + ".sql")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, string(sql))
}

func publishAccountForTest(t *testing.T, h *ApiHandler, r *gin.Engine, token string, server *httptest.Server) string {
	t.Helper()
	h.SocialHTTPClient = server.Client()
	account := createPostAccount(t, r, token, "mastodon", socialOrg)
	dbExec(t, h, "UPDATE uranus.social_account SET base_url=$1 WHERE uuid=$2", server.URL, account)
	return account
}

func publishData(t *testing.T, w *httptest.ResponseRecorder, status int) model.SocialPostPublish {
	t.Helper()
	assertSocialStatus(t, w, status)
	var envelope struct {
		Data         model.SocialPostPublish `json:"data"`
		ResponseType string                  `json:"response_type"`
	}
	if json.Unmarshal(w.Body.Bytes(), &envelope) != nil || envelope.ResponseType != "admin-publish-social-post" || envelope.Data.Results == nil {
		t.Fatal("invalid publish response")
	}
	return envelope.Data
}

// Exercise the asynchronous HTTP contract, then run the actual worker and read
// persisted outcomes through the API. No publisher runs inside the HTTP handler.
func publishAndProcess(t *testing.T, h *ApiHandler, r *gin.Engine, token, path string) model.SocialPostPublish {
	t.Helper()
	result := publishData(t, socialRequest(r, "POST", path, token, ""), http.StatusAccepted)
	if _, err := h.ProcessDueSocialTargets(context.Background(), 20); err != nil {
		t.Fatal(err)
	}
	post := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+result.PostUuid, token, ""))
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath+"?social_post_uuid="+result.PostUuid, token, ""))
	for i := range result.Results {
		response := &result.Results[i]
		for _, target := range post.Targets {
			if target.Uuid == response.TargetUuid {
				response.Status, response.RemotePostID, response.Error = target.Status, target.RemotePostID, target.Error
			}
		}
		for _, publication := range history {
			if publication.SocialPostTargetUuid == response.TargetUuid {
				response.PublicationUuid, response.ContentFingerprint = publication.Uuid, publication.ContentFingerprint
			}
		}
	}
	return result
}

func TestSocialPublishAuthenticationAndUUID(t *testing.T) {
	h := setupAuthTest(t)
	r := socialPostRouter(h)
	path := socialPostsPath + "/" + authTestUser + "/publish"
	for _, token := range []string{"", "invalid"} {
		assertSocialStatus(t, socialRequest(r, "POST", path, token, ""), 401)
	}
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/invalid/publish", socialToken(t, authTestUser), ""), 400)
	req := httptest.NewRequest("POST", path, nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: socialToken(t, authTestUser)})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assertSocialStatus(t, w, 403)
}

func TestSocialPublishPostgresSuccess(t *testing.T) {
	for _, withImage := range []bool{false, true} {
		t.Run(fmt.Sprint(withImage), func(t *testing.T) {
			h, r, token := publishDatabase(t)
			if !withImage {
				dbExec(t, h, "DELETE FROM uranus.pluto_image")
			}
			var expected model.RenderedPost
			var calls atomic.Int32
			var steps []string
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				steps = append(steps, req.URL.Path)
				switch req.URL.Path {
				case "/api/image/" + contentImage:
					if req.Header.Get("Authorization") != "" {
						t.Error("image download exposed token")
					}
					if expected.ImageURL != app.UranusInstance.Config.BaseApiUrl+req.URL.String() {
						t.Error("image differs from preview")
					}
					w.Header().Set("Content-Type", "image/png")
					png.Encode(w, image.NewRGBA(image.Rect(0, 0, 2, 2)))
				case "/api/v2/media":
					if req.ParseMultipartForm(1<<20) != nil {
						t.Error("invalid media")
						return
					}
					defer req.MultipartForm.RemoveAll()
					if expected.ImageAlt == nil || req.FormValue("description") != *expected.ImageAlt {
						t.Error("alt text differs from preview")
					}
					io.WriteString(w, `{"id":"77","url":"https://media.test/77"}`)
				case "/api/v1/statuses":
					calls.Add(1)
					if req.Header.Get("Authorization") != "Bearer secret-access" {
						t.Error("missing publishing credentials")
					}
					if req.ParseForm() != nil || req.Form.Get("status") != expected.Text || req.Form.Get("visibility") != "public" {
						t.Error("published text differs from preview")
					}
					if withImage && req.Form.Get("media_ids[]") != "77" {
						t.Error("missing media id")
					}
					io.WriteString(w, `{"id":"123","url":"https://example.test/@test/123"}`)
				default:
					t.Error("unexpected remote request")
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			app.UranusInstance.Config.BaseApiUrl = server.URL
			account := publishAccountForTest(t, h, r, token, server)
			post := previewPost(t, r, token, account)
			path := socialPostsPath + "/" + post.Uuid
			dbExec(t, h, "UPDATE uranus.social_post_target SET error='old error',scheduled_at='2026-09-23T12:00:00Z' WHERE uuid=$1", post.Targets[0].Uuid)
			expected = previewData(t, socialRequest(r, "POST", path+"/preview?lang=en", token, "")).Previews[0].RenderedPost
			before := socialPostData(t, socialRequest(r, "GET", path, token, ""))
			got := publishAndProcess(t, h, r, token, path+"/publish?lang=en")
			if got.PostUuid != post.Uuid || len(got.Results) != 1 || got.Results[0].Status != "published" || got.Results[0].Platform != "mastodon" {
				t.Fatal("incorrect publish result")
			}
			publication := publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+got.Results[0].PublicationUuid, token, ""))
			if publication.RenderedText == nil || *publication.RenderedText != expected.Text ||
				publication.RenderedImageURL == nil || *publication.RenderedImageURL != expected.ImageURL ||
				!reflect.DeepEqual(publication.RenderedImageAlt, expected.ImageAlt) || publication.Status != "published" {
				t.Fatal("history snapshot differs from the actual publisher payload")
			}
			after := socialPostData(t, socialRequest(r, "GET", path, token, ""))
			target := after.Targets[0]
			if target.Status != "published" || target.PublishedAt == nil || target.RemotePostID == nil || *target.RemotePostID != "123" || target.Error != nil ||
				!target.UpdatedAt.After(before.Targets[0].UpdatedAt) || (target.ScheduledAt == nil || !target.ScheduledAt.After(*before.Targets[0].ScheduledAt)) {
				t.Fatal("publication state was not persisted")
			}
			publishAndProcess(t, h, r, token, path+"/publish?lang=en")
			if calls.Load() != 1 {
				t.Fatal("published twice")
			}
			if withImage && (len(steps) != 3 || steps[0] != "/api/image/"+contentImage || steps[1] != "/api/v2/media") {
				t.Fatal("incorrect upload order")
			}
			storedSocialTokens(t, h, account, "secret-access", "secret-refresh")
		})
	}
}

func TestSocialPublishPostgresFailuresAndManualRepublish(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(422)
			io.WriteString(w, `{"error":"secret-access secret-refresh"}`)
			return
		}
		io.WriteString(w, `{"id":"123"}`)
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid
	got := publishAndProcess(t, h, r, token, path+"/publish")
	target := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
	if got.Results[0].Status != "failed" || target.Status != "failed" || target.PublishedAt != nil || target.RemotePostID != nil || target.Error == nil || *target.Error != "Mastodon returned HTTP 422" {
		t.Fatal("remote failure not persisted safely")
	}
	publishAndProcess(t, h, r, token, path+"/publish")
	if calls.Load() != 2 {
		t.Fatal("manual failed-target publishing did not run once")
	}
}

func TestSocialPublishPostgresInvalidAccounts(t *testing.T) {
	for _, tc := range []struct{ name, query, reason string }{
		{"disabled", "UPDATE uranus.social_account SET enabled=false WHERE uuid=$1", "disabled"},
		{"missing base", "UPDATE uranus.social_account SET base_url=NULL WHERE uuid=$1", "base_url"},
		{"missing token", "UPDATE uranus.social_account SET access_token=NULL WHERE uuid=$1", "access_token"},
		{"missing id", "UPDATE uranus.social_account SET remote_account_id=' ' WHERE uuid=$1", "remote_account_id"},
		{"foreign", "UPDATE uranus.social_account SET org_uuid='" + socialOtherOrg + "' WHERE uuid=$1", "another organization"},
		{"missing", "DELETE FROM uranus.social_account WHERE uuid=$1", "not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, r, token := publishDatabase(t)
			server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("invalid account made remote request") }))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			post := previewPost(t, r, token, account)
			// Corrupt fixtures explicitly to exercise defensive checks beyond the
			// normal Part 2 FK and account-organization restrictions.
			if tc.name == "foreign" {
				dbExec(t, h, "ALTER TABLE uranus.social_account DISABLE TRIGGER social_account_target_org")
			}
			if tc.name == "missing" {
				dbExec(t, h, "ALTER TABLE uranus.social_post_target DROP CONSTRAINT social_post_target_social_account_uuid_fkey")
			}
			if tc.name == "missing id" {
				dbExec(t, h, "ALTER TABLE uranus.social_account DROP CONSTRAINT social_account_remote_account_id_check")
			}
			dbExec(t, h, tc.query, account)
			got := publishAndProcess(t, h, r, token, socialPostsPath+"/"+post.Uuid+"/publish")
			if got.Results[0].Status != "failed" || got.Results[0].Error == nil || !strings.Contains(*got.Results[0].Error, tc.reason) {
				t.Fatal("incorrect invalid-account failure")
			}
		})
	}
}

func TestSocialPublishPostgresPermissionsAndSources(t *testing.T) {
	h, r, token := publishDatabase(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid + "/publish"
	otherToken := socialToken(t, socialOtherUser)
	assertSocialStatus(t, socialRequest(r, "POST", path, otherToken, ""), 403)
	dbExec(t, h, "INSERT INTO uranus.user_organization_link (user_uuid,org_uuid,permissions) VALUES ($1,$2,$3)", socialOtherUser, socialOrg, int64(app.UserPermEditOrg))
	dbExec(t, h, "INSERT INTO uranus.organization_member_link (user_uuid,org_uuid,has_joined) VALUES ($1,$2,false)", socialOtherUser, socialOrg)
	assertSocialStatus(t, socialRequest(r, "POST", path, otherToken, ""), 403)
	dbExec(t, h, "UPDATE uranus.organization_member_link SET has_joined=true WHERE user_uuid=$1", socialOtherUser)
	dbExec(t, h, "UPDATE uranus.user_organization_link SET permissions=0 WHERE user_uuid=$1", socialOtherUser)
	assertSocialStatus(t, socialRequest(r, "POST", path, otherToken, ""), 403)
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+socialOtherUser+"/publish", token, ""), 404)
	for _, source := range []string{socialOtherUser, contentOtherVenue} {
		dbExec(t, h, "UPDATE uranus.social_post SET source_type='venue',source_uuid=$1 WHERE uuid=$2", source, post.Uuid)
		got := publishAndProcess(t, h, r, token, path)
		if got.Results[0].Status != "failed" || got.Results[0].Error == nil {
			t.Fatal("invalid source did not create failed worker outcome")
		}
	}
	post = previewPost(t, r, token)
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/publish", token, ""), 400)
}

func TestSocialPublishPostgresStatesAndPartialSuccess(t *testing.T) {
	h, r, token := publishDatabase(t)
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); io.WriteString(w, `{"id":"123"}`) }))
	defer server.Close()
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	mastodon := publishAccountForTest(t, h, r, token, server)
	facebook := createPostAccount(t, r, token, "facebook", socialOrg)
	post := previewPost(t, r, token, mastodon, facebook)
	path := socialPostsPath + "/" + post.Uuid
	got := publishAndProcess(t, h, r, token, path+"/publish")
	if len(got.Results) != 2 {
		t.Fatal("missing target result")
	}
	for _, result := range got.Results {
		if result.Platform == "mastodon" && result.Status != "published" || result.Platform == "facebook" &&
			(result.Status != "failed" || result.Error == nil || *result.Error != "publishing is not implemented for platform facebook") {
			t.Fatal("partial success not represented")
		}
	}
	got = publishAndProcess(t, h, r, token, path+"/publish")
	for _, result := range got.Results {
		if result.Platform == "mastodon" && (result.Error != nil || result.Status != "published") {
			t.Fatal("duplicate target was not completed without a new publication")
		}
	}
	if calls.Load() != 1 {
		t.Fatal("published target was sent twice")
	}
	for _, state := range []string{"scheduled", "cancelled", "publishing"} {
		post = previewPost(t, r, token, mastodon)
		dbExec(t, h, "UPDATE uranus.social_post_target SET status=$1 WHERE uuid=$2", state, post.Targets[0].Uuid)
		got = publishData(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/publish", token, ""), 409)
		if got.Results[0].Status != state || calls.Load() != 1 {
			t.Fatal("forbidden state was published")
		}
	}
	// A published target does not block a newly added draft target.
	post = previewPost(t, r, token, mastodon, facebook)
	dbExec(t, h, "UPDATE uranus.social_post_target SET status='published' WHERE social_account_uuid=$1 AND social_post_uuid=$2", facebook, post.Uuid)
	publishAndProcess(t, h, r, token, socialPostsPath+"/"+post.Uuid+"/publish")
	if calls.Load() != 2 {
		t.Fatal("draft target was not processed alongside conflict")
	}
}

func TestSocialPublishPostgresConcurrent(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	entered, release := make(chan struct{}, 1), make(chan struct{})
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		entered <- struct{}{}
		select {
		case <-release:
			io.WriteString(w, `{"id":"123"}`)
		case <-req.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid
	other := *h
	otherRouter := socialPostRouter(&other)
	responses := make(chan *httptest.ResponseRecorder, 2)
	go func() { responses <- socialRequest(r, "POST", path+"/publish", token, "") }()
	go func() { responses <- socialRequest(otherRouter, "POST", path+"/publish", token, "") }()
	codes := map[int]int{}
	for range 2 {
		select {
		case response := <-responses:
			codes[response.Code]++
		case <-time.After(5 * time.Second):
			t.Fatal("enqueue request waited for publishing")
		}
	}
	if codes[202] != 1 || codes[409] != 1 || calls.Load() != 0 {
		t.Fatal("concurrent enqueue did not accept exactly one request without remote calls")
	}
	done := make(chan error, 1)
	go func() { _, err := other.ProcessDueSocialTargets(context.Background(), 20); done <- err }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not reach remote")
	}
	publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 409)
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"targets":[]}`), 409)
	assertSocialStatus(t, socialRequest(r, "DELETE", path, token, ""), 409)
	previewData(t, socialRequest(r, "POST", path+"/preview", token, ""))
	release <- struct{}{}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
	if calls.Load() != 1 || len(history) != 1 || history[0].Status != "published" || history[0].PublicationSource != "manual" {
		t.Fatal("concurrent enqueue created duplicate attempts")
	}
}

func TestSocialPublishPostgresUncertainAndDatabaseFailures(t *testing.T) {
	for _, failure := range []string{"remote", "database", "claim", "claim commit"} {
		t.Run(failure, func(t *testing.T) {
			h, r, token := publishDatabase(t)
			dbExec(t, h, "DELETE FROM uranus.pluto_image")
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if failure == "remote" {
					io.WriteString(w, `{"id":"secret-access"}`)
				} else {
					io.WriteString(w, `{"id":"123"}`)
				}
			}))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			post := previewPost(t, r, token, account)
			path := socialPostsPath + "/" + post.Uuid
			if failure != "remote" {
				state := "published"
				if strings.HasPrefix(failure, "claim") {
					state = "publishing"
				}
				dbExec(t, h, `CREATE FUNCTION uranus.fail_publish() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status='`+state+`' THEN RAISE EXCEPTION 'secret-access secret-refresh'; END IF; RETURN NEW; END $$`)
				trigger := "CREATE TRIGGER fail_publish AFTER UPDATE ON uranus.social_post_target"
				if failure == "claim commit" {
					trigger = "CREATE CONSTRAINT TRIGGER fail_publish AFTER UPDATE ON uranus.social_post_target DEFERRABLE INITIALLY DEFERRED"
				}
				dbExec(t, h, trigger+" FOR EACH ROW EXECUTE FUNCTION uranus.fail_publish()")
			}
			publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 202)
			_, workerErr := h.ProcessDueSocialTargets(context.Background(), 20)
			if strings.HasPrefix(failure, "claim") {
				if workerErr == nil || calls.Load() != 0 || socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0].Status != "scheduled" {
					t.Fatal("failed claim reached remote or persisted")
				}
				if len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))) != 0 {
					t.Fatal("failed claim left publication history")
				}
				return
			}
			if workerErr != nil {
				t.Fatal(workerErr)
			}
			got := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
			if len(got) != 1 || socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0].Status != "publishing" {
				t.Fatal("uncertain result did not retain claim")
			}
			publication := publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+got[0].Uuid, token, ""))
			want := "uncertain"
			if failure == "database" {
				want = "publishing"
			}
			if publication.Status != want {
				t.Fatal("uncertain or unfinalized publication was not retained")
			}
			publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 409)
			if calls.Load() != 1 {
				t.Fatal("uncertain publication repeated")
			}
		})
	}
}

func TestSocialPublishPostgresMigrationAndRenderParity(t *testing.T) {
	h, r, token := publishDatabase(t)
	account := createPostAccount(t, r, token, "mastodon", socialOrg)
	dbExec(t, h, "UPDATE uranus.social_account SET base_url='https://social.test' WHERE uuid=$1", account)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid
	for _, lang := range []string{"", "?lang=en", "?lang=unknown"} {
		expected := previewData(t, socialRequest(r, "POST", path+"/preview"+lang, token, "")).Previews[0].RenderedPost
		publishData(t, socialRequest(r, "POST", path+"/publish"+lang, token, ""), 202)
		work, result, err := h.claimScheduledSocialTarget(context.Background())
		if err != nil || !work.claimed || work.err != nil || !reflect.DeepEqual(work.rendered, expected) {
			t.Fatal("preview and worker render differently")
		}
		result.Status = "failed"
		if !h.finishSocialPublish(context.Background(), result) {
			t.Fatal("could not finalize render-parity attempt")
		}
	}
	dbExec(t, h, "UPDATE uranus.social_post_target SET status='publishing' WHERE uuid=$1", post.Targets[0].Uuid)
	for _, query := range []string{"DELETE FROM " + h.DbSchema + ".social_post WHERE uuid=$1", "DELETE FROM " + h.DbSchema + ".social_post_target WHERE social_post_uuid=$1"} {
		if _, err := h.DbPool.Exec(context.Background(), query, post.Uuid); err == nil {
			t.Fatal("deletion removed unresolved publication")
		}
	}
	sql, err := os.ReadFile("../migrations/202609250003_social_publish.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := h.DbPool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(context.Background(), strings.ReplaceAll(string(sql), "uranus.", h.DbSchema+"."))
	if err == nil {
		t.Error("rollback removed unresolved claim")
	}
	conn.Exec(context.Background(), "ROLLBACK")
	conn.Release()
	dbExec(t, h, "UPDATE uranus.social_post_target SET status='cancelled' WHERE uuid=$1", post.Targets[0].Uuid)
	before := socialPostData(t, socialRequest(r, "GET", path, token, ""))
	applySocialPublishMigration(t, h, "down")
	applySocialPublishMigration(t, h, "up")
	if !reflect.DeepEqual(before, socialPostData(t, socialRequest(r, "GET", path, token, ""))) {
		t.Fatal("migration changed existing target data")
	}
}

func TestSocialPublishPostgresCancellation(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	entered := make(chan struct{}, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		io.Copy(io.Discard, req.Body)
		entered <- struct{}{}
		<-req.Context().Done()
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 202)
	done := make(chan error, 1)
	go func() { _, err := h.ProcessDueSocialTargets(ctx, 20); done <- err }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("publish did not reach mock")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("remote request ignored cancellation")
	}
	target := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
	if target.Status != "publishing" || target.Error == nil {
		t.Fatal("cancelled outcome was not recorded")
	}
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
	if len(history) != 1 || history[0].Status != "uncertain" {
		t.Fatal("cancelled request did not record uncertain publication")
	}
}
