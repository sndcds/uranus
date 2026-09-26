package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSocialPublishPostgresQueueAndCurrentContent(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		if req.ParseForm() != nil || !strings.Contains(req.Form.Get("status"), "Edited after acceptance") ||
			req.Header.Get("Authorization") != "Bearer changed-test-credential" {
			t.Error("worker used content or credentials from enqueue time")
		}
		io.WriteString(w, `{"id":"1"}`)
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("POST", path+"/publish?lang=en", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	cancel() // A disconnected API caller cannot cancel a committed request.
	queued := publishData(t, w, 202)
	if len(queued.Results) != 1 || queued.Results[0].Status != "scheduled" || queued.Results[0].PublicationSource != "manual" ||
		queued.Results[0].PublicationUuid != "" || queued.Results[0].ContentFingerprint != nil || calls.Load() != 0 {
		t.Fatal("API executed publishing instead of queuing it")
	}
	if len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))) != 0 {
		t.Fatal("enqueue created a premature publication snapshot")
	}
	target := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
	if target.ScheduledAt == nil || target.ScheduledAt.After(time.Now()) || target.PublicationSource != "manual" || target.PublishLanguage == nil || *target.PublishLanguage != "en" {
		t.Fatal("manual request is not immediately available with its language preference")
	}
	publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 409)
	dbExec(t, h, "UPDATE uranus.event SET title='Edited after acceptance' WHERE uuid=$1", contentEvent)
	dbExec(t, h, "UPDATE uranus.social_account SET access_token='changed-test-credential' WHERE uuid=$1", account)
	processSocialTargetsForTest(t, h, 20, 1)
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
	if calls.Load() != 1 || len(history) != 1 || history[0].PublicationSource != "manual" || history[0].Status != "published" {
		t.Fatal("manual worker result was not recorded")
	}
}

func TestSocialPublishPostgresQueueFailure(t *testing.T) {
	for _, deferred := range []bool{false, true} {
		t.Run(map[bool]string{false: "statement", true: "commit"}[deferred], func(t *testing.T) {
			h, r, token := publishDatabase(t)
			account := createPostAccount(t, r, token, "facebook", socialOrg)
			post := previewPost(t, r, token, account)
			dbExec(t, h, `CREATE FUNCTION uranus.fail_enqueue() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'private queue detail'; END $$`)
			query := "CREATE TRIGGER fail_enqueue AFTER UPDATE ON uranus.social_post_target"
			if deferred {
				query = "CREATE CONSTRAINT TRIGGER fail_enqueue AFTER UPDATE ON uranus.social_post_target DEFERRABLE INITIALLY DEFERRED"
			}
			dbExec(t, h, query+" FOR EACH ROW EXECUTE FUNCTION uranus.fail_enqueue()")
			path := socialPostsPath + "/" + post.Uuid
			w := socialRequest(r, "POST", path+"/publish", token, "")
			assertSocialStatus(t, w, 500)
			if strings.Contains(w.Body.String(), "private queue") {
				t.Fatal("enqueue exposed database detail")
			}
			target := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
			if target.Status != "draft" || target.ScheduledAt != nil {
				t.Fatal("failed queue transaction left work for the worker")
			}
			processSocialTargetsForTest(t, h, 20, 0)
		})
	}
}

func TestSocialPublishPostgresQueueCancelAndReschedule(t *testing.T) {
	h, r, token := publishDatabase(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	for _, action := range []string{"cancel", "schedule"} {
		post := previewPost(t, r, token, account)
		path := socialPostsPath + "/" + post.Uuid
		publishData(t, socialRequest(r, "POST", path+"/publish?lang=en", token, ""), 202)
		assertSocialStatus(t, socialRequest(r, "POST", path+"/"+action, token, socialScheduleBody(time.Now().Add(time.Hour))), 200)
		processSocialTargetsForTest(t, h, 20, 0)
		if action == "schedule" {
			target := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
			if target.PublicationSource != "scheduled" || target.PublishLanguage != nil {
				t.Fatal("explicit schedule retained the old manual request options")
			}
		}
	}
}
