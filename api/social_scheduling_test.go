package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sndcds/uranus/model"
)

func socialScheduleBody(at time.Time) string {
	body, _ := json.Marshal(map[string]time.Time{"scheduled_at": at})
	return string(body)
}

func TestSocialSchedulingAuthenticationAndValidation(t *testing.T) {
	h := setupAuthTest(t)
	r, token := socialPostRouter(h), socialToken(t, authTestUser)
	for _, action := range []string{"schedule", "cancel"} {
		path := socialPostsPath + "/" + authTestUser + "/" + action
		for _, token := range []string{"", "invalid"} {
			assertSocialStatus(t, socialRequest(r, "POST", path, token, "{}"), 401)
		}
		assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/invalid/"+action, token, "{}"), 400)
		req := httptest.NewRequest("POST", path, strings.NewReader("{}"))
		req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assertSocialStatus(t, w, 403)
	}
	for _, body := range []string{
		"", "null", "[]", "{", "{}", `{"scheduled_at":null}`,
		`{"scheduled_at":"invalid"}`, `{"scheduled_at":"2027-01-01T18:00:00"}`,
		`{"scheduled_at":"2020-01-01T18:00:00Z"}`, `{"scheduled_at":123}`,
		`{"scheduled_at":"2027-01-01T18:00:00Z","targets":[]}`,
		`{"scheduled_at":"2027-01-01T18:00:00Z"} {}`,
		socialScheduleBody(time.Now().Add(-time.Minute)),
	} {
		assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+authTestUser+"/schedule", token, body), 400)
	}
}

func TestSocialSchedulingPostgresTransitions(t *testing.T) {
	h, r, token := publishDatabase(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	future := time.Now().Add(time.Hour).In(time.FixedZone("offset", 2*60*60)).Truncate(time.Second)
	for _, state := range []string{"draft", "failed", "scheduled", "published", "cancelled", "publishing"} {
		t.Run(state, func(t *testing.T) {
			post := previewPost(t, r, token, account)
			path := socialPostsPath + "/" + post.Uuid
			dbExec(t, h, "UPDATE uranus.social_post_target SET status=$1,scheduled_at=CURRENT_TIMESTAMP - INTERVAL '1 hour' WHERE uuid=$2", state, post.Targets[0].Uuid)
			if state == "published" {
				dbExec(t, h, "UPDATE uranus.social_post_target SET remote_post_id='1',published_at=CURRENT_TIMESTAMP WHERE uuid=$1", post.Targets[0].Uuid)
			}
			before := socialPostData(t, socialRequest(r, "GET", path, token, ""))
			w := socialRequest(r, "POST", path+"/schedule", token, socialScheduleBody(future))
			if state == "cancelled" || state == "publishing" {
				assertSocialStatus(t, w, 409)
				return
			}
			assertSocialStatus(t, w, 200)
			got := socialPostData(t, w)
			if got.Uuid != post.Uuid || len(got.Targets) != 1 || got.Targets[0].Uuid != before.Targets[0].Uuid ||
				got.Targets[0].Status != "scheduled" || got.Targets[0].ScheduledAt == nil || !got.Targets[0].ScheduledAt.Equal(future) {
				t.Fatalf("incorrect schedule response: %+v", got)
			}
			publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 409)
			assertSocialStatus(t, socialRequest(r, "POST", path+"/schedule", token, socialScheduleBody(future.Add(time.Hour))), 200)
			got = socialPostData(t, socialRequest(r, "GET", path, token, ""))
			if !got.Targets[0].ScheduledAt.Equal(future.Add(time.Hour)) {
				t.Fatal("reschedule did not persist")
			}
		})
	}
	if len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))) != 0 {
		t.Fatal("scheduling or blocked manual publish created a snapshot")
	}
}

func TestSocialCancelPostgresTransitions(t *testing.T) {
	h, r, token := publishDatabase(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	for _, state := range []string{"scheduled", "draft", "published", "failed", "publishing", "cancelled"} {
		t.Run(state, func(t *testing.T) {
			post := previewPost(t, r, token, account)
			path := socialPostsPath + "/" + post.Uuid
			dbExec(t, h, "UPDATE uranus.social_post_target SET status=$1,scheduled_at=CURRENT_TIMESTAMP WHERE uuid=$2", state, post.Targets[0].Uuid)
			before := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
			w := socialRequest(r, "POST", path+"/cancel", token, "")
			want := 409
			if state == "scheduled" {
				want = 200
			}
			assertSocialStatus(t, w, want)
			got := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
			if !got.ScheduledAt.Equal(*before.ScheduledAt) || (want == 200 && got.Status != "cancelled") || (want == 409 && got.Status != state) {
				t.Fatal("cancel changed unexpected state or schedule timestamp")
			}
		})
	}
}

func TestSocialSchedulingPostgresPermissionsAndMultipleTargets(t *testing.T) {
	h, r, token := publishDatabase(t)
	accounts := []string{
		createPostAccount(t, r, token, "facebook", socialOrg),
		createPostAccount(t, r, token, "instagram", socialOrg),
		createPostAccount(t, r, token, "mastodon", socialOrg),
	}
	post := previewPost(t, r, token, accounts...)
	path := socialPostsPath + "/" + post.Uuid
	body := socialScheduleBody(time.Now().Add(time.Hour))
	for _, action := range []string{"schedule", "cancel"} {
		assertSocialStatus(t, socialRequest(r, "POST", path+"/"+action, socialToken(t, socialOtherUser), body), 403)
		assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+socialOtherUser+"/"+action, token, body), 404)
	}
	// An unresolved target rejects the entire action, without partial changes.
	dbExec(t, h, "UPDATE uranus.social_post_target SET status='publishing' WHERE uuid=$1", post.Targets[1].Uuid)
	assertSocialStatus(t, socialRequest(r, "POST", path+"/schedule", token, body), 409)
	got := socialPostData(t, socialRequest(r, "GET", path, token, ""))
	if got.Targets[0].Status != "draft" {
		t.Fatal("conflict partially scheduled targets")
	}
	dbExec(t, h, "UPDATE uranus.social_post_target SET status='failed' WHERE uuid=$1", post.Targets[1].Uuid)
	dbExec(t, h, "UPDATE uranus.social_post_target SET status='cancelled' WHERE uuid=$1", post.Targets[2].Uuid)
	w := socialRequest(r, "POST", path+"/schedule", token, body)
	assertSocialStatus(t, w, 200)
	got = socialPostData(t, w)
	for i, target := range got.Targets {
		want := "scheduled"
		if i == 2 {
			want = "cancelled"
		}
		if target.Status != want {
			t.Fatal("schedule did not respect eligible target set")
		}
	}
	assertSocialStatus(t, socialRequest(r, "POST", path+"/cancel", token, ""), 200)
	for _, target := range socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets {
		if target.Status != "cancelled" {
			t.Fatal("cancel did not cover all scheduled targets")
		}
	}
	// No source rendering at schedule time, even for a missing source.
	w = socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOrg, "event", accounts[0]))
	assertSocialStatus(t, w, 201)
	missingSource := socialPostData(t, w)
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+missingSource.Uuid+"/schedule", token, body), 200)
	w = socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOrg, "event"))
	assertSocialStatus(t, w, 201)
	empty := socialPostData(t, w)
	for _, action := range []string{"schedule", "cancel"} {
		assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath+"/"+empty.Uuid+"/"+action, token, body), 409)
	}
	if len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))) != 0 {
		t.Fatal("scheduling wrote publication history")
	}
}

func scheduledTargetForTest(t *testing.T, h *ApiHandler, post model.SocialPost) {
	t.Helper()
	dbExec(t, h, "UPDATE uranus.social_post_target SET status='scheduled',publication_source='scheduled',publish_language=NULL,scheduled_at=CURRENT_TIMESTAMP - INTERVAL '1 minute' WHERE social_post_uuid=$1", post.Uuid)
}
