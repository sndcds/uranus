package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
	"github.com/sndcds/uranus/service"
)

func processSocialTargetsForTest(t *testing.T, h *ApiHandler, batchSize, want int) {
	t.Helper()
	got, err := h.ProcessDueSocialTargets(context.Background(), batchSize)
	if err != nil || got != want {
		t.Fatalf("processed %d, expected %d: %v", got, want, err)
	}
}

func TestSocialWorkerPostgresDueOrderAndBatch(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	var order []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		var targetUuid string
		err := h.DbPool.QueryRow(context.Background(), "SELECT social_post_target_uuid FROM "+h.DbSchema+".social_publication WHERE status='publishing'").Scan(&targetUuid)
		if err != nil {
			t.Error(err)
		}
		order = append(order, targetUuid)
		fmt.Fprintf(w, `{"id":"%d"}`, calls.Load())
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	var due []model.SocialPost
	for range 3 {
		post := previewPost(t, r, token, account)
		scheduledTargetForTest(t, h, post)
		due = append(due, post)
	}
	// Force equal due and creation times to exercise the UUID tie breaker.
	dbExec(t, h, "UPDATE uranus.social_post_target SET scheduled_at='2020-01-01T00:00:00Z',created_at='2020-01-01T00:00:00Z'")
	future := previewPost(t, r, token, account)
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+future.Uuid+"/schedule", token, socialScheduleBody(time.Now().Add(time.Hour))), 200)
	cancelled := previewPost(t, r, token, account)
	scheduledTargetForTest(t, h, cancelled)
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+cancelled.Uuid+"/cancel", token, ""), 200)
	for _, state := range []string{"failed", "draft", "published", "publishing"} {
		post := previewPost(t, r, token, account)
		dbExec(t, h, "UPDATE uranus.social_post_target SET status=$1,scheduled_at='2020-01-01T00:00:00Z' WHERE uuid=$2", state, post.Targets[0].Uuid)
	}
	processSocialTargetsForTest(t, h, 2, 2)
	processSocialTargetsForTest(t, h, 2, 1)
	processSocialTargetsForTest(t, h, 2, 0)
	want := []string{due[0].Targets[0].Uuid, due[1].Targets[0].Uuid, due[2].Targets[0].Uuid}
	if calls.Load() != 3 || !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected remote order: %v", order)
	}
	for _, post := range due {
		target := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, "")).Targets[0]
		if target.Status != "published" || target.PublishedAt == nil || target.ScheduledAt.Year() != 2020 || target.RemotePostID == nil {
			t.Fatal("incomplete scheduled publication state")
		}
	}
	for _, p := range publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, "")) {
		if p.PublicationSource != "scheduled" || p.Status != "published" || p.ContentFingerprint == nil {
			t.Fatal("worker attempt is missing its source, snapshot or outcome")
		}
	}
}

func TestSocialWorkerPostgresCurrentContentAndIdempotency(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	var expected model.RenderedPost
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		call := calls.Add(1)
		if req.ParseForm() != nil || req.Form.Get("status") != expected.Text {
			t.Error("remote request did not use current content")
		}
		var text, fingerprint, source string
		err := h.DbPool.QueryRow(context.Background(), "SELECT rendered_text,content_fingerprint,publication_source FROM "+h.DbSchema+".social_publication WHERE status='publishing'").Scan(&text, &fingerprint, &source)
		if err != nil || text != expected.Text || fingerprint != service.SocialContentFingerprint(expected) || source != "scheduled" {
			t.Error("snapshot was not committed before remote request")
		}
		fmt.Fprintf(w, `{"id":"%d"}`, call)
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid
	for i, title := range []string{"New title at execution", "Changed again", "New title at execution"} {
		assertSocialStatus(t, socialRequest(r, "POST", path+"/schedule", token, socialScheduleBody(time.Now().Add(time.Hour))), 200)
		// Source and account credentials are read at execution, after scheduling.
		dbExec(t, h, "UPDATE uranus.event SET title=$1 WHERE uuid=$2", title, contentEvent)
		expected = previewData(t, socialRequest(r, "POST", path+"/preview", token, "")).Previews[0].RenderedPost
		scheduledTargetForTest(t, h, post)
		processSocialTargetsForTest(t, h, 20, 1)
		wantCalls := int32(i + 1)
		if i == 2 {
			wantCalls = 2
		}
		if calls.Load() != wantCalls {
			t.Fatal("fingerprint idempotency failed")
		}
		target := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
		if target.Status != "published" || (i == 2 && (target.RemotePostID == nil || *target.RemotePostID != "1")) {
			t.Fatal("duplicate schedule did not restore its matching published result")
		}
	}
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
	if len(history) != 2 || history[0].ContentFingerprint == nil || history[1].ContentFingerprint == nil || *history[0].ContentFingerprint == *history[1].ContentFingerprint {
		t.Fatal("changed content did not produce distinct snapshots")
	}
	processSocialTargetsForTest(t, h, 20, 0)
}

func TestSocialWorkerPostgresLocalFailures(t *testing.T) {
	for _, failure := range []string{"disabled", "token removed", "event missing", "venue missing", "organization invalid", "oversized", "invalid URL", "instagram image", "instagram", "facebook", "bluesky"} {
		t.Run(failure, func(t *testing.T) {
			h, r, token := publishDatabase(t)
			if failure != "instagram" {
				dbExec(t, h, "DELETE FROM uranus.pluto_image")
			}
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				io.WriteString(w, `{"id":"1"}`)
			}))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			post := previewPost(t, r, token, account)
			scheduledTargetForTest(t, h, post)
			want := ""
			switch failure {
			case "disabled":
				dbExec(t, h, "UPDATE uranus.social_account SET enabled=false WHERE uuid=$1", account)
				want = "disabled"
			case "token removed":
				dbExec(t, h, "UPDATE uranus.social_account SET access_token=NULL WHERE uuid=$1", account)
				want = "access_token"
			case "event missing":
				dbExec(t, h, "DELETE FROM uranus.event WHERE uuid=$1", contentEvent)
				want = "source not found"
			case "venue missing":
				dbExec(t, h, "UPDATE uranus.social_post SET source_type='venue',source_uuid=$1 WHERE uuid=$2", socialOtherUser, post.Uuid)
				want = "source not found"
			case "organization invalid":
				dbExec(t, h, "UPDATE uranus.social_post SET source_type='organization',source_uuid=$1 WHERE uuid=$2", socialOtherOrg, post.Uuid)
				want = "source not found"
			case "oversized":
				dbExec(t, h, "UPDATE uranus.event SET title=$1 WHERE uuid=$2", strings.Repeat("long ", 500), contentEvent)
				want = "limit"
			case "invalid URL":
				dbExec(t, h, "UPDATE uranus.event SET source_link='javascript:invalid' WHERE uuid=$1", contentEvent)
				want = "URL"
			case "instagram image":
				dbExec(t, h, "UPDATE uranus.social_account SET platform='instagram' WHERE uuid=$1", account)
				want = "image"
			case "facebook", "instagram", "bluesky":
				dbExec(t, h, "UPDATE uranus.social_account SET platform=$1 WHERE uuid=$2", failure, account)
				want = "publishing is not implemented for platform " + failure
			}
			processSocialTargetsForTest(t, h, 20, 1)
			processSocialTargetsForTest(t, h, 20, 0)
			target := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, "")).Targets[0]
			history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
			if calls.Load() != 0 || target.Status != "failed" || len(history) != 1 || history[0].Status != "failed" ||
				history[0].PublicationSource != "scheduled" || history[0].Error == nil || !strings.Contains(*history[0].Error, want) {
				t.Fatalf("incorrect local failure: target=%+v history=%+v", target, history)
			}
		})
	}
}

func TestSocialWorkerPostgresRemoteOutcomes(t *testing.T) {
	for _, status := range []int{401, 403, 404, 422, 429, 500, 502, 200} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			h, r, token := publishDatabase(t)
			dbExec(t, h, "DELETE FROM uranus.pluto_image")
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if calls.Add(1) == 1 {
					w.WriteHeader(status)
					io.WriteString(w, `{"error":"private remote response"}`)
					return
				}
				io.WriteString(w, `{"id":"2"}`)
			}))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			first := previewPost(t, r, token, account)
			second := previewPost(t, r, token, account)
			scheduledTargetForTest(t, h, first)
			scheduledTargetForTest(t, h, second)
			processSocialTargetsForTest(t, h, 20, 2)
			processSocialTargetsForTest(t, h, 20, 0)
			history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
			wantTarget, wantPublication := "failed", "failed"
			if status >= 500 || status == 200 {
				wantTarget, wantPublication = "publishing", "uncertain"
			}
			target := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+first.Uuid, token, "")).Targets[0]
			if calls.Load() != 2 || target.Status != wantTarget || len(history) != 2 || history[0].Status != wantPublication || history[1].Status != "published" {
				t.Fatal("worker stopped after a target failure or retried it")
			}
			if history[0].Error == nil || strings.Contains(*history[0].Error, "private remote") {
				t.Fatal("remote response was not sanitized")
			}
		})
	}
}

func TestSocialWorkerPostgresDatabaseFailures(t *testing.T) {
	for _, failure := range []string{"claim", "commit", "finalization"} {
		t.Run(failure, func(t *testing.T) {
			h, r, token := publishDatabase(t)
			dbExec(t, h, "DELETE FROM uranus.pluto_image")
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				io.WriteString(w, `{"id":"1"}`)
			}))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			post := previewPost(t, r, token, account)
			scheduledTargetForTest(t, h, post)
			state := "publishing"
			if failure == "finalization" {
				state = "published"
			}
			dbExec(t, h, `CREATE FUNCTION uranus.fail_worker() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status='`+state+`' THEN RAISE EXCEPTION 'private database detail'; END IF; RETURN NEW; END $$`)
			query := "CREATE TRIGGER fail_worker AFTER UPDATE ON uranus.social_post_target"
			if failure == "commit" {
				query = "CREATE CONSTRAINT TRIGGER fail_worker AFTER UPDATE ON uranus.social_post_target DEFERRABLE INITIALLY DEFERRED"
			}
			dbExec(t, h, query+" FOR EACH ROW EXECUTE FUNCTION uranus.fail_worker()")
			_, err := h.ProcessDueSocialTargets(context.Background(), 20)
			history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
			target := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, "")).Targets[0]
			if failure != "finalization" {
				if err == nil || calls.Load() != 0 || len(history) != 0 || target.Status != "scheduled" || strings.Contains(err.Error(), "private database") {
					t.Fatal("failed claim reached remote or left partial state")
				}
				return
			}
			if err != nil || calls.Load() != 1 || len(history) != 1 || history[0].Status != "publishing" || target.Status != "publishing" {
				t.Fatal("failed finalization lost recoverable state")
			}
			processSocialTargetsForTest(t, h, 20, 0)
			dbExec(t, h, "DROP TRIGGER fail_worker ON uranus.social_post_target")
			publicationData(t, socialRequest(r, "POST", socialPublicationsPath+"/"+history[0].Uuid+"/reconcile", token,
				`{"outcome":"published","remote_post_id":"1","confirm_inactive":true}`))
			processSocialTargetsForTest(t, h, 20, 0)
			if calls.Load() != 1 {
				t.Fatal("recoverable success published twice")
			}
		})
	}
}

func TestSocialWorkerConfigAndCancelledContext(t *testing.T) {
	config := app.DefaultConfig()
	h := &ApiHandler{Config: &config}
	if config.SocialWorkerInterval != 30 || config.SocialWorkerBatchSize != 20 {
		t.Fatal("unexpected worker defaults")
	}
	for _, interval := range []int{-1, 0, 14, 3601} {
		config.SocialWorkerInterval = interval
		if h.RunSocialWorker(context.Background(), true) == nil {
			t.Fatal("invalid interval accepted")
		}
	}
	config = app.DefaultConfig()
	for _, size := range []int{-1, 0, 101} {
		config.SocialWorkerBatchSize = size
		if h.RunSocialWorker(context.Background(), true) == nil {
			t.Fatal("invalid batch size accepted")
		}
	}
	config = app.DefaultConfig()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.RunSocialWorker(ctx, false); err != nil {
		t.Fatal(err)
	}
}

func TestSocialWorkerPostgresCurrentAccountAndScheduleOwnership(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		if req.Header.Get("Authorization") != "Bearer replacement-test-token" {
			t.Error("worker used stale credentials")
		}
		io.WriteString(w, `{"id":"1"}`)
	}))
	defer server.Close()
	account := createPostAccount(t, r, token, "mastodon", socialOrg)
	post := previewPost(t, r, token, account)
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/schedule", token, socialScheduleBody(time.Now().Add(time.Hour))), 200)
	h.SocialHTTPClient = server.Client()
	dbExec(t, h, "UPDATE uranus.social_account SET access_token='replacement-test-token',base_url=$1 WHERE uuid=$2", server.URL, account)
	// Leaving the organization after authorization does not revoke its schedule.
	dbExec(t, h, "DELETE FROM uranus.organization_member_link WHERE user_uuid=$1", authTestUser)
	scheduledTargetForTest(t, h, post)
	processSocialTargetsForTest(t, h, 20, 1)
	var status, source string
	if err := h.DbPool.QueryRow(context.Background(), "SELECT status,publication_source FROM "+h.DbSchema+".social_publication").Scan(&status, &source); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || status != "published" || source != "scheduled" {
		t.Fatal("worker required departed user's authorization")
	}
}

func TestSocialWorkerPostgresManualAndLegacyIdempotency(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint(legacy), func(t *testing.T) {
			h, r, token := publishDatabase(t)
			dbExec(t, h, "DELETE FROM uranus.pluto_image")
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				io.WriteString(w, `{"id":"1"}`)
			}))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			post := previewPost(t, r, token, account)
			path := socialPostsPath + "/" + post.Uuid
			if legacy {
				dbExec(t, h, "UPDATE uranus.social_post_target SET status='published',remote_post_id='1',published_at=CURRENT_TIMESTAMP WHERE uuid=$1", post.Targets[0].Uuid)
			} else {
				publishAndProcess(t, h, r, token, path+"/publish")
			}
			before := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
			assertSocialStatus(t, socialRequest(r, "POST", path+"/schedule", token, socialScheduleBody(time.Now().Add(time.Hour))), 200)
			scheduledTargetForTest(t, h, post)
			processSocialTargetsForTest(t, h, 20, 1)
			after := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
			want := int32(1)
			if legacy {
				want = 0
			}
			if calls.Load() != want || after.Status != "published" || !after.PublishedAt.Equal(*before.PublishedAt) || *after.RemotePostID != *before.RemotePostID {
				t.Fatal("scheduled duplicate changed original publication or reached remote")
			}
			if len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))) != int(want) {
				t.Fatal("duplicate created new history")
			}
		})
	}
}

func TestSocialWorkerPostgresSkipsLockedPosts(t *testing.T) {
	h, r, token := publishDatabase(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	first := previewPost(t, r, token, account)
	second := previewPost(t, r, token, account)
	scheduledTargetForTest(t, h, first)
	scheduledTargetForTest(t, h, second)
	tx, err := h.DbPool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), "SELECT uuid FROM "+h.DbSchema+".social_post WHERE uuid=$1 FOR UPDATE", first.Uuid); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	count, err := h.ProcessDueSocialTargets(ctx, 20)
	if err != nil || count != 1 {
		t.Fatal("locked post prevented processing other due work")
	}
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
	if len(history) != 1 || history[0].SocialPostUuid != second.Uuid {
		t.Fatal("worker failed to skip the locked post")
	}
}

func TestSocialWorkerPostgresMultipleTargetsAndCascade(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		io.WriteString(w, `{"id":"1"}`)
	}))
	defer server.Close()
	mastodon := publishAccountForTest(t, h, r, token, server)
	facebook := createPostAccount(t, r, token, "facebook", socialOrg)
	post := previewPost(t, r, token, facebook, mastodon)
	scheduledTargetForTest(t, h, post)
	processSocialTargetsForTest(t, h, 20, 2)
	got := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, ""))
	states := map[string]string{}
	for _, target := range got.Targets {
		states[target.SocialAccountUuid] = target.Status
	}
	if calls.Load() != 1 || states[facebook] != "failed" || states[mastodon] != "published" {
		t.Fatal("one target failure stopped other targets of the post")
	}
	otherAccount := createPostAccount(t, r, token, "facebook", socialOtherOrg)
	w := socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOtherOrg, "organization", otherAccount))
	assertSocialStatus(t, w, 201)
	other := socialPostData(t, w)
	scheduledTargetForTest(t, h, other)
	dbExec(t, h, "DELETE FROM uranus.organization WHERE uuid=$1", socialOtherOrg)
	processSocialTargetsForTest(t, h, 20, 0)
	var dangling bool
	if err := h.DbPool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM "+h.DbSchema+".social_post_target WHERE social_post_uuid=$1)", other.Uuid).Scan(&dangling); err != nil || dangling {
		t.Fatal("organization cascade left a dangling schedule")
	}
}

func TestSocialWorkerPostgresUnavailableDatabase(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		io.WriteString(w, `{"id":"1"}`)
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	scheduledTargetForTest(t, h, post)
	config := h.DbPool.Config()
	config.ConnConfig.Host, config.ConnConfig.Port = "127.0.0.1", 1
	config.ConnConfig.ConnectTimeout = time.Second
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	worker := *h
	worker.DbPool = pool
	count, err := worker.ProcessDueSocialTargets(context.Background(), 20)
	if err == nil || count != 0 || calls.Load() != 0 {
		t.Fatal("unavailable database authorized a remote call")
	}
	target := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, "")).Targets[0]
	if target.Status != "scheduled" || len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))) != 0 {
		t.Fatal("database outage left partial attempt state")
	}
}
