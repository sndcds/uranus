package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func socialPostDatabase(t *testing.T) (*ApiHandler, *gin.Engine, string) {
	t.Helper()
	h, r, token := socialDatabase(t)
	applySocialPostMigration(t, h, "up")
	h.RegisterSocialPostRoutes(r.Group("/api/admin", app.JWTMiddleware))
	return h, r, token
}

func applySocialPostMigration(t *testing.T, h *ApiHandler, direction string) {
	t.Helper()
	sql, err := os.ReadFile("../migrations/202609250002_social_post." + direction + ".sql")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, string(sql))
}

func socialPostBody(org, source string, accounts ...string) string {
	targets := make([]socialPostTargetInput, 0, len(accounts))
	for _, account := range accounts {
		targets = append(targets, socialPostTargetInput{SocialAccountUuid: account})
	}
	body, _ := json.Marshal(gin.H{"org_uuid": org, "source_type": source, "source_uuid": authTestUser, "targets": targets})
	return string(body)
}

func socialPostData(t *testing.T, w *httptest.ResponseRecorder) model.SocialPost {
	t.Helper()
	var response struct {
		Data model.SocialPost `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Targets == nil {
		t.Fatal("targets must be an array, including when empty")
	}
	return response.Data
}

func socialPostListData(t *testing.T, w *httptest.ResponseRecorder) []model.SocialPost {
	t.Helper()
	assertSocialStatus(t, w, 200)
	var response struct {
		Data struct {
			Posts []model.SocialPost `json:"posts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Posts == nil {
		t.Fatal("posts must be an array, including when empty")
	}
	return response.Data.Posts
}

func createPostAccount(t *testing.T, r *gin.Engine, token, platform, org string) string {
	t.Helper()
	w := socialRequest(r, "POST", socialAccountsPath, token, socialBody(platform, org, platform))
	assertSocialStatus(t, w, 201)
	return socialAccountData(t, w).Uuid
}

func TestSocialPostsPostgresCRUD(t *testing.T) {
	h, r, token := socialPostDatabase(t)
	captureSocialLogs(t)
	facebook := createPostAccount(t, r, token, "facebook", socialOrg)
	instagram := createPostAccount(t, r, token, "instagram", socialOrg)
	mastodon := createPostAccount(t, r, token, "mastodon", socialOrg)
	if len(socialPostListData(t, socialRequest(r, "GET", socialPostsPath, token, ""))) != 0 {
		t.Fatal("new list is not empty")
	}
	for _, source := range []string{"event", "venue", "organization"} {
		w := socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOrg, source, facebook, instagram))
		assertSocialStatus(t, w, 201)
		post := socialPostData(t, w)
		if !app.ValidUUID(post.Uuid) || post.OrgUuid != socialOrg || post.SourceType != source || post.SourceUuid != authTestUser ||
			post.CreatedBy == nil || *post.CreatedBy != authTestUser || post.CreatedAt.IsZero() || post.UpdatedAt.IsZero() || len(post.Targets) != 2 {
			t.Fatalf("unexpected post: %+v", post)
		}
		for _, target := range post.Targets {
			if !app.ValidUUID(target.Uuid) || target.Status != "draft" || target.ScheduledAt != nil || target.PublishedAt != nil ||
				target.RemotePostID != nil || target.Error != nil || target.CreatedAt.IsZero() || target.UpdatedAt.IsZero() {
				t.Fatalf("unexpected target: %+v", target)
			}
		}
		path := socialPostsPath + "/" + post.Uuid
		w = socialRequest(r, "GET", path, token, "")
		assertSocialStatus(t, w, 200)
		if !reflect.DeepEqual(socialPostData(t, w), post) {
			t.Fatal("GET differs from creation response")
		}
		posts := socialPostListData(t, socialRequest(r, "GET", socialPostsPath, token, ""))
		if len(posts) != 1 || !reflect.DeepEqual(posts[0], post) {
			t.Fatal("list differs from creation response")
		}
		// Every stored status and all metadata survive target reconciliation.
		for _, status := range []string{"draft", "scheduled", "published", "failed", "cancelled"} {
			dbExec(t, h, `UPDATE uranus.social_post_target SET status=$1,
				scheduled_at='2027-01-02T03:04:05Z', published_at='2027-01-03T03:04:05Z',
				remote_post_id='remote-42', error='saved error', updated_at='2026-01-01T00:00:00Z'
				WHERE social_post_uuid=$2 AND social_account_uuid=$3`, status, post.Uuid, facebook)
			w = socialRequest(r, "GET", path, token, "")
			assertSocialStatus(t, w, 200)
			before := socialPostData(t, w)
			var retained model.SocialPostTarget
			for _, target := range before.Targets {
				if target.SocialAccountUuid == facebook {
					retained = target
				}
			}
			w = socialRequest(r, "PUT", path, token, socialPostBody(socialOrg, "venue", facebook, mastodon))
			assertSocialStatus(t, w, 200)
			updated := socialPostData(t, w)
			if updated.SourceType != "venue" || len(updated.Targets) != 2 || !updated.UpdatedAt.After(before.UpdatedAt) ||
				!updated.CreatedAt.Equal(post.CreatedAt) || *updated.CreatedBy != authTestUser {
				t.Fatal("post update failed")
			}
			for _, target := range updated.Targets {
				if target.SocialAccountUuid == facebook {
					if !reflect.DeepEqual(target, retained) {
						t.Fatal("retained target metadata changed")
					}
				} else if target.SocialAccountUuid != mastodon || target.Status != "draft" {
					t.Fatal("target addition/removal failed")
				}
			}
		}
		w = socialRequest(r, "GET", path, token, "")
		assertSocialStatus(t, w, 200)
		before := socialPostData(t, w)
		w = socialRequest(r, "PUT", path, token, `{"source_uuid":"`+socialOtherOrg+`"}`)
		assertSocialStatus(t, w, 200)
		if updated := socialPostData(t, w); updated.SourceUuid != socialOtherOrg || !reflect.DeepEqual(updated.Targets, before.Targets) {
			t.Fatal("omitted targets changed")
		}
		// Published targets have no special deletion rule in Part 2.
		dbExec(t, h, "UPDATE uranus.social_post_target SET status='published' WHERE social_post_uuid=$1", post.Uuid)
		assertSocialStatus(t, socialRequest(r, "DELETE", path, token, ""), 200)
		for _, method := range []string{"GET", "PUT", "DELETE"} {
			assertSocialStatus(t, socialRequest(r, method, path, token, "{}"), 404)
		}
		var count int
		if err := h.DbPool.QueryRow(context.Background(), "SELECT count(*) FROM "+h.DbSchema+".social_post_target WHERE social_post_uuid=$1", post.Uuid).Scan(&count); err != nil || count != 0 {
			t.Fatalf("target cascade failed: count=%d err=%v", count, err)
		}
	}

	// Both omitted and empty targets are valid at creation; [] clears on PUT.
	for _, body := range []string{
		`{"org_uuid":"` + socialOrg + `","source_type":"event","source_uuid":"` + authTestUser + `"}`,
		socialPostBody(socialOrg, "event"), socialPostBody(socialOrg, "event", facebook),
	} {
		w := socialRequest(r, "POST", socialPostsPath, token, body)
		assertSocialStatus(t, w, 201)
		post := socialPostData(t, w)
		path := socialPostsPath + "/" + post.Uuid
		w = socialRequest(r, "PUT", path, token, `{"targets":[]}`)
		assertSocialStatus(t, w, 200)
		if len(socialPostData(t, w).Targets) != 0 {
			t.Fatal("empty targets did not clear the list")
		}
		assertSocialStatus(t, socialRequest(r, "DELETE", path, token, ""), 200)
	}
}

func TestSocialPostsPostgresTargetValidation(t *testing.T) {
	h, r, token := socialPostDatabase(t)
	captureSocialLogs(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	disabled := createPostAccount(t, r, token, "instagram", socialOrg)
	foreign := createPostAccount(t, r, token, "mastodon", socialOtherOrg)
	assertSocialStatus(t, socialRequest(r, "PUT", socialAccountsPath+"/"+disabled, token, `{"enabled":false}`), 200)
	w := socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOrg, "event", account))
	assertSocialStatus(t, w, 201)
	post := socialPostData(t, w)
	path := socialPostsPath + "/" + post.Uuid
	for _, targets := range [][]string{{account, account}, {authTestUser}, {disabled}, {foreign}, {account, foreign}} {
		body := socialPostBody(socialOrg, "venue", targets...)
		assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath, token, body), 400)
		assertSocialStatus(t, socialRequest(r, "PUT", path, token, body), 400)
		w = socialRequest(r, "GET", path, token, "")
		assertSocialStatus(t, w, 200)
		if !reflect.DeepEqual(post, socialPostData(t, w)) || len(socialPostListData(t, socialRequest(r, "GET", socialPostsPath, token, ""))) != 1 {
			t.Fatal("invalid target write was not rolled back")
		}
	}
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"org_uuid":"`+socialOtherOrg+`"}`), 400)
	accountPath := socialAccountsPath + "/" + account
	assertSocialStatus(t, socialRequest(r, "DELETE", accountPath, token, ""), 409)
	assertSocialStatus(t, socialRequest(r, "PUT", accountPath, token, `{"org_uuid":"`+socialOtherOrg+`"}`), 409)
	storedSocialTokens(t, h, account, "secret-access", "secret-refresh")
	// Disabling an account does not delete existing targets. Omitted targets can
	// still be kept while editing the source; explicitly selected accounts validate.
	assertSocialStatus(t, socialRequest(r, "PUT", accountPath, token, `{"enabled":false}`), 200)
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"source_type":"venue"}`), 200)
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, socialPostBody(socialOrg, "venue", account)), 400)
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"targets":[]}`), 200)
	assertSocialStatus(t, socialRequest(r, "PUT", accountPath, token, `{"org_uuid":"`+socialOtherOrg+`"}`), 200)
	assertSocialStatus(t, socialRequest(r, "DELETE", accountPath, token, ""), 200)
}

func TestSocialPostsPostgresAuthorization(t *testing.T) {
	h, r, token := socialPostDatabase(t)
	w := socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOrg, "event"))
	assertSocialStatus(t, w, 201)
	path := socialPostsPath + "/" + socialPostData(t, w).Uuid
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOtherOrg, "event")), 201)
	otherToken := socialToken(t, socialOtherUser)
	assertDenied := func() {
		t.Helper()
		for _, method := range []string{"GET", "PUT", "DELETE"} {
			assertSocialStatus(t, socialRequest(r, method, path, otherToken, "{}"), 403)
		}
		assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath, otherToken, socialPostBody(socialOrg, "event")), 403)
		for _, suffix := range []string{"", "?org_uuid=" + socialOrg} {
			if len(socialPostListData(t, socialRequest(r, "GET", socialPostsPath+suffix, otherToken, ""))) != 0 {
				t.Fatal("unauthorized post in list")
			}
		}
	}
	assertDenied()
	dbExec(t, h, "INSERT INTO uranus.user_organization_link (user_uuid,org_uuid,permissions) VALUES ($1,$2,$3)", socialOtherUser, socialOrg, int64(app.UserPermEditOrg))
	dbExec(t, h, "INSERT INTO uranus.organization_member_link (user_uuid,org_uuid,has_joined) VALUES ($1,$2,false)", socialOtherUser, socialOrg)
	assertDenied()
	dbExec(t, h, "UPDATE uranus.organization_member_link SET has_joined=true WHERE user_uuid=$1", socialOtherUser)
	dbExec(t, h, "UPDATE uranus.user_organization_link SET permissions=0 WHERE user_uuid=$1", socialOtherUser)
	assertDenied()
	dbExec(t, h, "UPDATE uranus.user_organization_link SET permissions=$1 WHERE user_uuid=$2", int64(app.UserPermEditOrg), socialOtherUser)
	assertSocialStatus(t, socialRequest(r, "GET", path, otherToken, ""), 200)
	if len(socialPostListData(t, socialRequest(r, "GET", socialPostsPath, token, ""))) != 2 ||
		len(socialPostListData(t, socialRequest(r, "GET", socialPostsPath+"?org_uuid="+socialOrg, token, ""))) != 1 ||
		len(socialPostListData(t, socialRequest(r, "GET", socialPostsPath, otherToken, ""))) != 1 {
		t.Fatal("organization list/filter failed")
	}
	// Unknown organizations have no permission grant, matching Part 1.
	assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOtherUser, "event")), 403)
}

func TestSocialPostsPostgresMigration(t *testing.T) {
	h, r, token := socialPostDatabase(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	w := socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOrg, "event", account))
	assertSocialStatus(t, w, 201)
	post := socialPostData(t, w)
	for _, query := range []string{
		fmt.Sprintf("UPDATE %s.social_post SET source_type='template'", h.DbSchema),
		fmt.Sprintf("UPDATE %s.social_post SET org_uuid='%s'", h.DbSchema, socialOtherUser),
		fmt.Sprintf("UPDATE %s.social_post_target SET status='pending'", h.DbSchema),
		fmt.Sprintf("UPDATE %s.social_post_target SET social_post_uuid='%s'", h.DbSchema, socialOtherUser),
		fmt.Sprintf("UPDATE %s.social_post_target SET social_account_uuid='%s'", h.DbSchema, socialOtherUser),
		fmt.Sprintf("INSERT INTO %s.social_post_target (uuid,social_post_uuid,social_account_uuid) VALUES ('%s','%s','%s')", h.DbSchema, socialOtherUser, post.Uuid, account),
		fmt.Sprintf("DELETE FROM %s.social_account WHERE uuid='%s'", h.DbSchema, account),
		fmt.Sprintf("UPDATE %s.social_account SET org_uuid='%s' WHERE uuid='%s'", h.DbSchema, socialOtherOrg, account),
	} {
		if _, err := h.DbPool.Exec(context.Background(), query); err == nil {
			t.Fatalf("migration accepted invalid change: %s", query)
		}
	}
	// A deleted creator leaves the post intact, following nullable created_by.
	dbExec(t, h, `DELETE FROM uranus."user" WHERE uuid=$1`, authTestUser)
	var createdBy *string
	if err := h.DbPool.QueryRow(context.Background(), "SELECT created_by FROM "+h.DbSchema+".social_post WHERE uuid=$1", post.Uuid).Scan(&createdBy); err != nil || createdBy != nil {
		t.Fatalf("creator deletion failed: %v", err)
	}
	dbExec(t, h, "DELETE FROM uranus.organization WHERE uuid=$1", socialOrg)
	for _, table := range []string{"social_post", "social_post_target", "social_account"} {
		var count int
		if err := h.DbPool.QueryRow(context.Background(), "SELECT count(*) FROM "+h.DbSchema+"."+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("organization cascade failed for %s: count=%d err=%v", table, count, err)
		}
	}
	applySocialPostMigration(t, h, "down")
	applySocialPostMigration(t, h, "up")
	if len(socialPostListData(t, socialRequest(r, "GET", socialPostsPath, token, ""))) != 0 {
		t.Fatal("migration roundtrip failed")
	}
}

func TestSocialPostsPostgresRollbackAndSecretErrors(t *testing.T) {
	h, r, token := socialPostDatabase(t)
	captureSocialLogs(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	other := createPostAccount(t, r, token, "instagram", socialOrg)
	w := socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOrg, "event", account))
	assertSocialStatus(t, w, 201)
	post := socialPostData(t, w)
	path := socialPostsPath + "/" + post.Uuid
	for _, deferred := range []bool{false, true} {
		dbExec(t, h, `CREATE FUNCTION uranus.fail_post_write() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN RAISE EXCEPTION 'secret-access secret-refresh'; END $$`)
		trigger := "CREATE TRIGGER fail_post AFTER INSERT OR UPDATE OR DELETE ON uranus.social_post"
		if deferred {
			trigger = "CREATE CONSTRAINT TRIGGER fail_post AFTER INSERT OR UPDATE OR DELETE ON uranus.social_post DEFERRABLE INITIALLY DEFERRED"
		}
		dbExec(t, h, trigger+" FOR EACH ROW EXECUTE FUNCTION uranus.fail_post_write()")
		assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath, token, socialPostBody(socialOrg, "event", other)), 500)
		assertSocialStatus(t, socialRequest(r, "PUT", path, token, socialPostBody(socialOrg, "venue", other)), 500)
		assertSocialStatus(t, socialRequest(r, "DELETE", path, token, ""), 500)
		w = socialRequest(r, "GET", path, token, "")
		assertSocialStatus(t, w, 200)
		if !reflect.DeepEqual(post, socialPostData(t, w)) || len(socialPostListData(t, socialRequest(r, "GET", socialPostsPath, token, ""))) != 1 {
			t.Fatal("failed write was not rolled back")
		}
		dbExec(t, h, "DROP TRIGGER fail_post ON uranus.social_post")
		dbExec(t, h, "DROP FUNCTION uranus.fail_post_write()")
	}
}
