package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
	"github.com/sndcds/uranus/service"
)

const socialPublicationsPath = "/api/admin/social/publications"

func applySocialPublicationMigration(t *testing.T, h *ApiHandler, direction string) {
	t.Helper()
	sql, err := os.ReadFile("../migrations/202609260001_social_publication." + direction + ".sql")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, string(sql))
}

func publicationData(t *testing.T, w *httptest.ResponseRecorder) model.SocialPublication {
	t.Helper()
	assertSocialStatus(t, w, 200)
	var envelope struct {
		Data model.SocialPublication `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func publicationList(t *testing.T, w *httptest.ResponseRecorder) []model.SocialPublication {
	t.Helper()
	assertSocialStatus(t, w, 200)
	var envelope struct {
		Data struct {
			Publications []model.SocialPublication `json:"publications"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Publications == nil {
		t.Fatal("history must be an array")
	}
	return envelope.Data.Publications
}

func TestSocialPublicationsPostgresHistoryAndIdempotency(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		call := calls.Add(1)
		// The exact attempt must be committed before the remote mutation begins.
		var status, text string
		if err := h.DbPool.QueryRow(context.Background(), "SELECT status,rendered_text FROM "+h.DbSchema+".social_publication ORDER BY created_at DESC,uuid DESC LIMIT 1").Scan(&status, &text); err != nil {
			t.Error(err)
		}
		if req.ParseForm() != nil || status != "publishing" || text != req.Form.Get("status") {
			t.Error("remote payload differs from committed snapshot")
		}
		fmt.Fprintf(w, `{"id":"%d"}`, call)
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid
	expected := previewData(t, socialRequest(r, "POST", path+"/preview", token, "")).Previews[0].RenderedPost
	if len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))) != 0 {
		t.Fatal("preview wrote publication history")
	}
	first := publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 200).Results[0]
	a := publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+first.PublicationUuid, token, ""))
	if a.Status != "published" || a.ContentFingerprint == nil || *a.ContentFingerprint != service.SocialContentFingerprint(expected) ||
		a.RenderedText == nil || *a.RenderedText != expected.Text || a.RenderedImageURL == nil || *a.RenderedImageURL != expected.ImageURL ||
		!reflect.DeepEqual(a.RenderedImageAlt, expected.ImageAlt) || a.RemotePostID == nil || *a.RemotePostID != "1" || a.FinishedAt == nil {
		t.Fatal("incomplete publication history")
	}
	publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 409)
	dbExec(t, h, "UPDATE uranus.event SET title='Geändertes Konzert' WHERE uuid=$1", contentEvent)
	second := publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 200).Results[0]
	if second.ContentFingerprint == nil || *second.ContentFingerprint == *a.ContentFingerprint || second.PublicationUuid == a.Uuid {
		t.Fatal("new content did not create a new attempt")
	}
	dbExec(t, h, "UPDATE uranus.event SET title='Konzert' WHERE uuid=$1", contentEvent)
	publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 409)
	if calls.Load() != 2 {
		t.Fatal("duplicate content reached remote")
	}
	if got := publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+a.Uuid, token, "")); !reflect.DeepEqual(a, got) {
		t.Fatal("historical snapshot changed")
	}
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath+"?social_post_uuid="+post.Uuid, token, ""))
	if len(history) != 2 || history[0].Uuid != a.Uuid || history[1].Uuid != second.PublicationUuid {
		t.Fatal("history ordering is not stable")
	}
	target := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
	if target.RemotePostID == nil || *target.RemotePostID != "2" || target.PublishedAt == nil {
		t.Fatal("target does not reflect latest success")
	}
	assertSocialStatus(t, socialRequest(r, "DELETE", path, token, ""), 409)
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"targets":[]}`), 409)
	assertSocialStatus(t, socialRequest(r, "DELETE", socialAccountsPath+"/"+account, token, ""), 409)
}

func TestSocialPublicationsPostgresFailedRetry(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(422)
			io.WriteString(w, `{"error":"secret-access secret-refresh raw-response"}`)
			return
		}
		io.WriteString(w, `{"id":"123"}`)
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	path := socialPostsPath + "/" + post.Uuid + "/publish"
	first := publishData(t, socialRequest(r, "POST", path, token, ""), 207).Results[0]
	failed := publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+first.PublicationUuid, token, ""))
	if failed.Status != "failed" || failed.Error == nil || *failed.Error != "Mastodon returned HTTP 422" {
		t.Fatal("failure was not recorded safely")
	}
	second := publishData(t, socialRequest(r, "POST", path, token, ""), 200).Results[0]
	if second.PublicationUuid == first.PublicationUuid || *second.ContentFingerprint != *first.ContentFingerprint {
		t.Fatal("retry did not create separate history for the same content")
	}
	if got := publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+first.PublicationUuid, token, "")); !reflect.DeepEqual(failed, got) {
		t.Fatal("retry overwrote the failed attempt")
	}
	var rows string
	if err := h.DbPool.QueryRow(context.Background(), "SELECT jsonb_agg(to_jsonb(p))::text FROM "+h.DbSchema+".social_publication p").Scan(&rows); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret-access", "secret-refresh", "raw-response", "access_token", "Authorization"} {
		if strings.Contains(rows, secret) {
			t.Fatal("publication contains credentials or raw response")
		}
	}
}

func TestSocialPublicationsPostgresReconcile(t *testing.T) {
	for _, outcome := range []string{"published", "failed"} {
		t.Run(outcome, func(t *testing.T) {
			h, r, token := publishDatabase(t)
			dbExec(t, h, "DELETE FROM uranus.pluto_image")
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if calls.Add(1) == 1 {
					w.WriteHeader(503)
					return
				}
				io.WriteString(w, `{"id":"124"}`)
			}))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			post := previewPost(t, r, token, account)
			path := socialPostsPath + "/" + post.Uuid
			first := publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 207).Results[0]
			publicationPath := socialPublicationsPath + "/" + first.PublicationUuid
			uncertain := publicationData(t, socialRequest(r, "GET", publicationPath, token, ""))
			if uncertain.Status != "uncertain" || uncertain.FinishedAt == nil || first.Status != "publishing" {
				t.Fatal("uncertainty not represented")
			}
			publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 409)
			body := `{"outcome":"failed"}`
			if outcome == "published" {
				body = `{"outcome":"published","remote_post_id":"123"}`
			}
			resolved := publicationData(t, socialRequest(r, "POST", publicationPath+"/reconcile", token, body))
			if resolved.Status != outcome || resolved.ReconciledAt == nil || calls.Load() != 1 || !reflect.DeepEqual(uncertain.RenderedText, resolved.RenderedText) {
				t.Fatal("reconciliation changed snapshot or made remote call")
			}
			target := socialPostData(t, socialRequest(r, "GET", path, token, "")).Targets[0]
			if target.Status != outcome || !reflect.DeepEqual(target.RemotePostID, resolved.RemotePostID) || !reflect.DeepEqual(target.PublishedAt, resolved.FinishedAt) && outcome == "published" {
				t.Fatal("reconciliation did not atomically repair target")
			}
			assertSocialStatus(t, socialRequest(r, "POST", publicationPath+"/reconcile", token, body), 409)
			if outcome == "failed" {
				if target.RemotePostID != nil || target.PublishedAt != nil {
					t.Fatal("failed reconciliation retained remote state")
				}
				next := publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 200).Results[0]
				if next.PublicationUuid == first.PublicationUuid {
					t.Fatal("retry reused publication")
				}
			} else {
				publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 409)
			}
		})
	}
}

func TestSocialPublicationsAuthentication(t *testing.T) {
	h := setupAuthTest(t)
	r := socialPostRouter(h)
	for _, route := range []struct{ method, path string }{
		{"GET", socialPublicationsPath},
		{"GET", socialPublicationsPath + "/" + authTestUser},
		{"POST", socialPublicationsPath + "/" + authTestUser + "/reconcile"},
	} {
		for _, token := range []string{"", "invalid"} {
			assertSocialStatus(t, socialRequest(r, route.method, route.path, token, `{"outcome":"failed"}`), 401)
		}
	}
	for _, route := range []struct{ method, path string }{
		{"GET", socialPublicationsPath + "/invalid"},
		{"POST", socialPublicationsPath + "/invalid/reconcile"},
	} {
		assertSocialStatus(t, socialRequest(r, route.method, route.path, socialToken(t, authTestUser), `{"outcome":"failed"}`), 400)
	}
}

func TestSocialPublicationsPostgresFiltersAndPermissions(t *testing.T) {
	h, r, token := publishDatabase(t)
	if len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))) != 0 {
		t.Fatal("new history is not empty")
	}
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	post := previewPost(t, r, token, account)
	first := publishData(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/publish", token, ""), 207).Results[0]
	for _, filter := range []string{
		"org_uuid=" + socialOrg, "social_post_uuid=" + post.Uuid,
		"social_post_target_uuid=" + post.Targets[0].Uuid, "status=failed",
		"org_uuid=" + socialOrg + "&social_post_uuid=" + post.Uuid + "&status=failed",
	} {
		list := publicationList(t, socialRequest(r, "GET", socialPublicationsPath+"?"+filter, token, ""))
		if len(list) != 1 || list[0].Uuid != first.PublicationUuid {
			t.Fatal("filter lost matching publication")
		}
	}
	for _, filter := range []string{"org_uuid=" + socialOtherOrg, "status=published", "social_post_uuid=" + socialOtherOrg, "social_post_target_uuid=" + socialOtherOrg} {
		if len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath+"?"+filter, token, ""))) != 0 {
			t.Fatal("filter included unrelated publication")
		}
	}
	for _, filter := range []string{"org_uuid=invalid", "social_post_uuid=invalid", "social_post_target_uuid=invalid", "status=draft", "status="} {
		assertSocialStatus(t, socialRequest(r, "GET", socialPublicationsPath+"?"+filter, token, ""), 400)
	}
	path := socialPublicationsPath + "/" + first.PublicationUuid
	otherToken := socialToken(t, socialOtherUser)
	dbExec(t, h, "INSERT INTO uranus.user_organization_link (user_uuid,org_uuid,permissions) VALUES ($1,$2,$3)",
		socialOtherUser, socialOtherOrg, int64(app.UserPermEditOrg))
	dbExec(t, h, "INSERT INTO uranus.organization_member_link (user_uuid,org_uuid,has_joined) VALUES ($1,$2,true)",
		socialOtherUser, socialOtherOrg)
	otherAccount := createPostAccount(t, r, otherToken, "facebook", socialOtherOrg)
	body, err := json.Marshal(map[string]any{
		"org_uuid": socialOtherOrg, "source_type": "organization", "source_uuid": socialOtherOrg,
		"targets": []socialPostTargetInput{{SocialAccountUuid: otherAccount}},
	})
	if err != nil {
		t.Fatal(err)
	}
	created := socialRequest(r, "POST", socialPostsPath, otherToken, string(body))
	assertSocialStatus(t, created, 201)
	otherPost := socialPostData(t, created)
	otherPublication := publishData(t, socialRequest(r, "POST", socialPostsPath+"/"+otherPost.Uuid+"/publish", otherToken, ""), 207).Results[0]
	ownHistory := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, otherToken, ""))
	if len(ownHistory) != 1 || ownHistory[0].Uuid != otherPublication.PublicationUuid {
		t.Fatal("history did not isolate organizations for an authorized member")
	}
	assertSocialStatus(t, socialRequest(r, "GET", path, otherToken, ""), 403)
	assertSocialStatus(t, socialRequest(r, "POST", path+"/reconcile", otherToken, `{"outcome":"failed"}`), 403)
	if len(publicationList(t, socialRequest(r, "GET", socialPublicationsPath+"?org_uuid="+socialOrg, otherToken, ""))) != 0 {
		t.Fatal("list leaked foreign organization history")
	}
	// Membership and permission remain independent requirements, as in posts.
	dbExec(t, h, "UPDATE uranus.organization_member_link SET has_joined=false WHERE user_uuid=$1", authTestUser)
	assertSocialStatus(t, socialRequest(r, "GET", path, token, ""), 403)
	dbExec(t, h, "UPDATE uranus.organization_member_link SET has_joined=true WHERE user_uuid=$1", authTestUser)
	dbExec(t, h, "UPDATE uranus.user_organization_link SET permissions=0 WHERE user_uuid=$1", authTestUser)
	assertSocialStatus(t, socialRequest(r, "GET", path, token, ""), 403)
	assertSocialStatus(t, socialRequest(r, "GET", socialPublicationsPath+"/"+socialOtherUser, token, ""), 404)
	assertSocialStatus(t, socialRequest(r, "POST", socialPublicationsPath+"/"+socialOtherUser+"/reconcile", token, `{"outcome":"failed"}`), 404)
}

func TestSocialPublicationsPostgresReconcileValidationAndRace(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, `{"id":"secret-access"}`)
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	first := publishData(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/publish", token, ""), 207).Results[0]
	path := socialPublicationsPath + "/" + first.PublicationUuid + "/reconcile"
	for _, body := range []string{
		"null", "{", "{}", `{"outcome":"uncertain"}`, `{"outcome":"published"}`,
		`{"outcome":"published","remote_post_id":null}`, `{"outcome":"published","remote_post_id":"secret-access"}`,
		`{"outcome":"published","remote_post_id":"123\n"}`,
		`{"outcome":"failed","remote_post_id":"123"}`, `{"outcome":"failed","remote_post_id":null}`,
		`{"outcome":"failed","access_token":"secret-access"}`, `{"outcome":"failed"} {}`,
	} {
		assertSocialStatus(t, socialRequest(r, "POST", path, token, body), 400)
	}
	other := *h
	otherRouter := socialPostRouter(&other)
	responses := make(chan *httptest.ResponseRecorder, 2)
	go func() {
		responses <- socialRequest(r, "POST", path, token, `{"outcome":"published","remote_post_id":"123"}`)
	}()
	go func() { responses <- socialRequest(otherRouter, "POST", path, token, `{"outcome":"failed"}`) }()
	codes := map[int]int{}
	for range 2 {
		response := <-responses
		codes[response.Code]++
	}
	if codes[200] != 1 || codes[409] != 1 {
		t.Fatal("more than one reconciliation won")
	}
	publication := publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+first.PublicationUuid, token, ""))
	target := socialPostData(t, socialRequest(r, "GET", socialPostsPath+"/"+post.Uuid, token, "")).Targets[0]
	if publication.Status != target.Status || !reflect.DeepEqual(publication.RemotePostID, target.RemotePostID) {
		t.Fatal("reconciliation race split target and publication state")
	}
}

func TestSocialPublicationsPostgresFinalizationRecovery(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		io.WriteString(w, `{"id":"123"}`)
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	dbExec(t, h, `CREATE FUNCTION uranus.fail_finish() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status='published' THEN RAISE EXCEPTION 'secret-access'; END IF; RETURN NEW; END $$`)
	dbExec(t, h, "CREATE TRIGGER fail_finish BEFORE UPDATE ON uranus.social_post_target FOR EACH ROW EXECUTE FUNCTION uranus.fail_finish()")
	first := publishData(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/publish", token, ""), 500).Results[0]
	path := socialPublicationsPath + "/" + first.PublicationUuid
	publication := publicationData(t, socialRequest(r, "GET", path, token, ""))
	if publication.Status != "publishing" || publication.FinishedAt != nil || publication.RemotePostID != nil {
		t.Fatal("failed finalization partially committed")
	}
	assertSocialStatus(t, socialRequest(r, "POST", path+"/reconcile", token, `{"outcome":"published","remote_post_id":"123"}`), 409)
	assertSocialStatus(t, socialRequest(r, "POST", path+"/reconcile", token,
		`{"outcome":"published","remote_post_id":"123","confirm_inactive":true}`), 500)
	if got := publicationData(t, socialRequest(r, "GET", path, token, "")); !reflect.DeepEqual(publication, got) {
		t.Fatal("failed reconciliation partially committed publication")
	}
	dbExec(t, h, "DROP TRIGGER fail_finish ON uranus.social_post_target")
	resolved := publicationData(t, socialRequest(r, "POST", path+"/reconcile", token, `{"outcome":"published","remote_post_id":"123","confirm_inactive":true}`))
	if resolved.Status != "published" || resolved.ReconciledAt == nil || calls.Load() != 1 {
		t.Fatal("persisted claim is not recoverable")
	}
	first.Status = "failed"
	if h.finishSocialPublish(context.Background(), first) {
		t.Fatal("late finalizer overwrote reconciled outcome")
	}
	if got := publicationData(t, socialRequest(r, "GET", path, token, "")); !reflect.DeepEqual(resolved, got) {
		t.Fatal("late finalizer changed history")
	}
	publishData(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/publish", token, ""), 409)
}

func TestSocialPublicationsPostgresLastSuccessAndMultipleTargets(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if calls.Add(1) == 2 {
			w.WriteHeader(422)
			return
		}
		io.WriteString(w, `{"id":"123"}`)
	}))
	defer server.Close()
	firstAccount := publishAccountForTest(t, h, r, token, server)
	response := socialRequest(r, "POST", socialAccountsPath, token, socialBody("mastodon", socialOrg, "second-account"))
	assertSocialStatus(t, response, 201)
	secondAccount := socialAccountData(t, response).Uuid
	dbExec(t, h, "UPDATE uranus.social_account SET base_url=$1 WHERE uuid=$2", server.URL, secondAccount)
	post := previewPost(t, r, token, firstAccount, secondAccount)
	path := socialPostsPath + "/" + post.Uuid
	publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 207)
	history := publicationList(t, socialRequest(r, "GET", socialPublicationsPath, token, ""))
	if len(history) != 2 || history[0].Status != "published" || history[1].Status != "failed" || history[0].SocialPostTargetUuid == history[1].SocialPostTargetUuid {
		t.Fatal("targets did not retain independent outcomes")
	}
	// Same content may be sent to another account; only its failed attempt retries.
	publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 207)
	if calls.Load() != 3 {
		t.Fatal("partial retry sent already published target")
	}
	before := socialPostData(t, socialRequest(r, "GET", path, token, ""))
	dbExec(t, h, "UPDATE uranus.event SET title='New content' WHERE uuid=$1", contentEvent)
	// A definite failure of a new revision retains each target's last success.
	h.SocialHTTPClient = &http.Client{Transport: publicationRejectTransport{}}
	publishData(t, socialRequest(r, "POST", path+"/publish", token, ""), 207)
	after := socialPostData(t, socialRequest(r, "GET", path, token, ""))
	for i, target := range after.Targets {
		if target.Status != "failed" || target.Error == nil || !reflect.DeepEqual(target.PublishedAt, before.Targets[i].PublishedAt) ||
			!reflect.DeepEqual(target.RemotePostID, before.Targets[i].RemotePostID) {
			t.Fatal("new failed attempt erased last successful publication")
		}
	}
}

type publicationRejectTransport struct{}

func (publicationRejectTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 422, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}"))}, nil
}

func TestSocialPublicationsPostgresLegacyPublished(t *testing.T) {
	for _, legacyClaim := range []bool{false, true} {
		t.Run(fmt.Sprint(legacyClaim), func(t *testing.T) {
			h, r, token := contentDatabase(t)
			applySocialPublishMigration(t, h, "up")
			server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("unknown legacy published content must not be repeated")
			}))
			defer server.Close()
			account := publishAccountForTest(t, h, r, token, server)
			post := previewPost(t, r, token, account)
			status := "published"
			if legacyClaim {
				status = "publishing"
			}
			dbExec(t, h, "UPDATE uranus.social_post_target SET status=$1 WHERE uuid=$2", status, post.Targets[0].Uuid)
			applySocialPublicationMigration(t, h, "up")
			if legacyClaim {
				publicationData(t, socialRequest(r, "POST", socialPublicationsPath+"/"+post.Targets[0].Uuid+"/reconcile", token,
					`{"outcome":"published","remote_post_id":"123"}`))
			}
			publishData(t, socialRequest(r, "POST", socialPostsPath+"/"+post.Uuid+"/publish", token, ""), 409)
		})
	}
}

func TestSocialPublicationsPostgresSnapshotAfterClaim(t *testing.T) {
	h, r, token := publishDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	var expected model.RenderedPost
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.ParseForm() != nil || req.Form.Get("status") != expected.Text {
			t.Error("publisher re-rendered content after claim")
		}
		io.WriteString(w, `{"id":"123"}`)
	}))
	defer server.Close()
	account := publishAccountForTest(t, h, r, token, server)
	post := previewPost(t, r, token, account)
	result := model.SocialPostPublish{PostUuid: post.Uuid}
	gc := contentContext(authTestUser)
	work, txErr := h.claimSocialPublish(gc, post.Uuid, &result)
	if txErr != nil || len(work) != 1 {
		t.Fatal("claim failed")
	}
	expected = work[0].rendered
	dbExec(t, h, "UPDATE uranus.event SET title='Changed after claim' WHERE uuid=$1", contentEvent)
	published, err := h.publishSocialTarget(context.Background(), work[0])
	if err != nil {
		t.Fatal(err)
	}
	result.Results[0].Status, result.Results[0].RemotePostID = "published", &published.RemotePostID
	if !h.finishSocialPublish(context.Background(), result.Results[0]) {
		t.Fatal("finalization failed")
	}
	publication := publicationData(t, socialRequest(r, "GET", socialPublicationsPath+"/"+result.Results[0].PublicationUuid, token, ""))
	if publication.RenderedText == nil || *publication.RenderedText != expected.Text || publication.ContentFingerprint == nil ||
		*publication.ContentFingerprint != service.SocialContentFingerprint(expected) {
		t.Fatal("stored content or fingerprint changed after claim")
	}
}
