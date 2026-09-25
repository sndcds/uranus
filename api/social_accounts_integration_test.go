package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

const (
	socialOrg       = "01994126-6680-7000-8000-000000000002"
	socialOtherOrg  = "01994126-6680-7000-8000-000000000003"
	socialOtherUser = "01994126-6680-7000-8000-000000000004"
)

func socialDatabase(t *testing.T) (*ApiHandler, *gin.Engine, string) {
	t.Helper()
	dsn := os.Getenv("URANUS_SOCIAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set URANUS_SOCIAL_TEST_DATABASE_URL to run PostgreSQL social account tests")
	}
	h := setupAuthTest(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	id, err := grains_uuid.Uuidv7String()
	if err != nil {
		t.Fatal(err)
	}
	h.DbPool, h.DbSchema = pool, "social_test_"+strings.ReplaceAll(id, "-", "")
	dbExec(t, h, "CREATE SCHEMA "+h.DbSchema)
	t.Cleanup(func() { dbExec(t, h, "DROP SCHEMA "+h.DbSchema+" CASCADE") })
	// Real organization DDL includes a PostGIS point. Only an explicitly supplied
	// disposable database is touched, never the application's configuration.
	dbExec(t, h, "CREATE EXTENSION IF NOT EXISTS postgis")
	for _, table := range []string{"user", "organization", "user_organization_link", "organization_member_link"} {
		ddl, err := os.ReadFile("../ddl/" + table + ".ddl")
		if err != nil {
			t.Fatal(err)
		}
		dbExec(t, h, strings.Split(string(ddl), "-- Indices")[0])
	}
	applySocialMigration(t, h, "up")
	sql, err := os.ReadFile("../sql/admin-get-user-org-permissions.sql")
	if err != nil {
		t.Fatal(err)
	}
	app.UranusInstance.SqlGetUserOrgPermissions = strings.ReplaceAll(string(sql), "{{schema}}", h.DbSchema)
	app.UranusInstance.Config.DebugLevel = 1
	for _, user := range []string{authTestUser, socialOtherUser} {
		dbExec(t, h, `INSERT INTO uranus."user" (uuid,email,password_hash,is_active) VALUES ($1,$2,'unused',true)`, user, user+"@example.test")
	}
	for _, org := range []string{socialOrg, socialOtherOrg} {
		dbExec(t, h, "INSERT INTO uranus.organization (uuid,name) VALUES ($1,'Social test')", org)
		dbExec(t, h, "INSERT INTO uranus.user_organization_link (user_uuid,org_uuid,permissions) VALUES ($1,$2,$3)",
			authTestUser, org, int64(app.UserPermCombinationAdmin))
		dbExec(t, h, "INSERT INTO uranus.organization_member_link (user_uuid,org_uuid,has_joined) VALUES ($1,$2,true)",
			authTestUser, org)
	}
	return h, socialRouter(h), socialToken(t, authTestUser)
}

func applySocialMigration(t *testing.T, h *ApiHandler, direction string) {
	t.Helper()
	sql, err := os.ReadFile("../migrations/202609250001_social_account." + direction + ".sql")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, string(sql))
}

func socialBody(platform, org, remote string) string {
	return `{"org_uuid":"` + org + `","platform":"` + platform + `","name":"@test",
		"remote_account_id":"` + remote + `","remote_account_name":"test",
		"access_token":"secret-access","refresh_token":"secret-refresh",
		"token_expires_at":"2027-01-02T03:04:05Z"}`
}

func socialAccountData(t *testing.T, w *httptest.ResponseRecorder) model.SocialAccount {
	t.Helper()
	var response struct {
		Data model.SocialAccount `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.Data
}

func socialListData(t *testing.T, w *httptest.ResponseRecorder) []model.SocialAccount {
	t.Helper()
	assertSocialStatus(t, w, 200)
	var response struct {
		Data struct {
			Accounts []model.SocialAccount `json:"accounts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Accounts == nil {
		t.Fatal("list must return an array, including when empty")
	}
	return response.Data.Accounts
}

func storedSocialTokens(t *testing.T, h *ApiHandler, id, access, refresh string) {
	t.Helper()
	var gotAccess, gotRefresh string
	err := h.DbPool.QueryRow(context.Background(), "SELECT COALESCE(access_token,''), COALESCE(refresh_token,'') FROM "+h.DbSchema+".social_account WHERE uuid=$1", id).Scan(&gotAccess, &gotRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if gotAccess != access || gotRefresh != refresh {
		t.Fatal("stored credentials differ from expected values")
	}
}

func TestSocialAccountsPostgresCRUD(t *testing.T) {
	h, r, token := socialDatabase(t)
	// Instagram is created first, without any Facebook connection or base URL.
	for _, platform := range []string{"instagram", "facebook", "mastodon", "bluesky"} {
		t.Run(platform, func(t *testing.T) {
			body := socialBody(platform, socialOrg, "same-platform-id")
			if platform == "mastodon" {
				body = strings.TrimSuffix(body, "}") + `,"base_url":"https://social.example"}`
			}
			w := socialRequest(r, "POST", socialAccountsPath, token, body)
			assertSocialStatus(t, w, 201)
			a := socialAccountData(t, w)
			if !app.ValidUUID(a.Uuid) || a.Platform != platform || a.OrgUuid != socialOrg ||
				!a.Enabled || !a.HasAccessToken || !a.HasRefreshToken || a.TokenExpiresAt == nil ||
				a.CreatedAt.IsZero() || a.UpdatedAt.IsZero() ||
				(platform != "mastodon" && a.BaseURL != nil) {
				t.Fatalf("unexpected metadata: %+v", a)
			}
			storedSocialTokens(t, h, a.Uuid, "secret-access", "secret-refresh")
			path := socialAccountsPath + "/" + a.Uuid
			assertSocialStatus(t, socialRequest(r, "GET", path, token, ""), 200)
			assertSocialStatus(t, socialRequest(r, "GET", socialAccountsPath, token, ""), 200)
			w = socialRequest(r, "PUT", path, token, `{"name":"@changed","enabled":false}`)
			assertSocialStatus(t, w, 200)
			updated := socialAccountData(t, w)
			if updated.Name != "@changed" || updated.Enabled || !updated.UpdatedAt.After(a.UpdatedAt) {
				t.Fatal("metadata update failed")
			}
			storedSocialTokens(t, h, a.Uuid, "secret-access", "secret-refresh")
			assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"access_token":"secret-access-new"}`), 200)
			storedSocialTokens(t, h, a.Uuid, "secret-access-new", "secret-refresh")
			assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"refresh_token":"secret-refresh-new"}`), 200)
			storedSocialTokens(t, h, a.Uuid, "secret-access-new", "secret-refresh-new")
			w = socialRequest(r, "PUT", path, token, `{"access_token":null,"refresh_token":"","token_expires_at":null,"base_url":null,"remote_account_name":null}`)
			assertSocialStatus(t, w, 200)
			cleared := socialAccountData(t, w)
			if cleared.HasAccessToken || cleared.HasRefreshToken || cleared.TokenExpiresAt != nil || cleared.BaseURL != nil || cleared.RemoteAccountName != nil {
				t.Fatal("explicit credential/metadata clearing failed")
			}
			storedSocialTokens(t, h, a.Uuid, "", "")
			assertSocialStatus(t, socialRequest(r, "DELETE", path, token, ""), 200)
			for _, method := range []string{"GET", "PUT", "DELETE"} {
				assertSocialStatus(t, socialRequest(r, method, path, token, "{}"), 404)
			}
		})
	}
	if len(socialListData(t, socialRequest(r, "GET", socialAccountsPath, token, ""))) != 0 {
		t.Fatal("deleted accounts remain")
	}
	// All optional connection fields may be absent or explicitly null.
	for _, optional := range []string{"", `,"access_token":null,"refresh_token":null,"token_expires_at":null,"base_url":null`} {
		w := socialRequest(r, "POST", socialAccountsPath, token,
			`{"org_uuid":"`+socialOrg+`","platform":"instagram","name":"minimal","remote_account_id":"minimal"`+optional+"}")
		assertSocialStatus(t, w, 201)
		a := socialAccountData(t, w)
		if !a.Enabled || a.HasAccessToken || a.HasRefreshToken || a.BaseURL != nil || a.TokenExpiresAt != nil {
			t.Fatal("incorrect defaults for optional fields")
		}
		assertSocialStatus(t, socialRequest(r, "DELETE", socialAccountsPath+"/"+a.Uuid, token, ""), 200)
	}
}

func TestSocialAccountsPostgresIdentityAndAuthorization(t *testing.T) {
	h, r, token := socialDatabase(t)
	w := socialRequest(r, "POST", socialAccountsPath, token, socialBody("instagram", socialOrg, "42"))
	assertSocialStatus(t, w, 201)
	id := socialAccountData(t, w).Uuid
	path := socialAccountsPath + "/" + id
	assertSocialStatus(t, socialRequest(r, "POST", socialAccountsPath, token, socialBody("instagram", socialOrg, "42")), 409)
	// Same remote ID is permitted on a different platform or in another org.
	assertSocialStatus(t, socialRequest(r, "POST", socialAccountsPath, token, socialBody("facebook", socialOrg, "42")), 201)
	assertSocialStatus(t, socialRequest(r, "POST", socialAccountsPath, token, socialBody("instagram", socialOtherOrg, "42")), 201)
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"platform":"facebook","access_token":"secret-access-new"}`), 409)
	storedSocialTokens(t, h, id, "secret-access", "secret-refresh")
	if len(socialListData(t, socialRequest(r, "GET", socialAccountsPath, token, ""))) != 3 ||
		len(socialListData(t, socialRequest(r, "GET", socialAccountsPath+"?org_uuid="+socialOrg, token, ""))) != 2 {
		t.Fatal("list/filter returned incorrect organizations")
	}
	otherToken := socialToken(t, socialOtherUser)
	assertDenied := func() {
		t.Helper()
		for _, method := range []string{"GET", "PUT", "DELETE"} {
			assertSocialStatus(t, socialRequest(r, method, path, otherToken, `{"name":"unauthorized","access_token":"secret-access-new"}`), 403)
		}
		assertSocialStatus(t, socialRequest(r, "POST", socialAccountsPath, otherToken, socialBody("bluesky", socialOrg, "did:example")), 403)
		for _, suffix := range []string{"", "?org_uuid=" + socialOrg} {
			if len(socialListData(t, socialRequest(r, "GET", socialAccountsPath+suffix, otherToken, ""))) != 0 {
				t.Fatal("unauthorized organization leaked into list")
			}
		}
	}
	assertDenied()
	// A link alone does not authorize an unaccepted invitation.
	dbExec(t, h, "INSERT INTO uranus.user_organization_link (user_uuid,org_uuid,permissions) VALUES ($1,$2,$3)", socialOtherUser, socialOrg, int64(app.UserPermEditOrg))
	dbExec(t, h, "INSERT INTO uranus.organization_member_link (user_uuid,org_uuid,has_joined) VALUES ($1,$2,false)", socialOtherUser, socialOrg)
	assertDenied()
	dbExec(t, h, "UPDATE uranus.organization_member_link SET has_joined=true WHERE user_uuid=$1", socialOtherUser)
	dbExec(t, h, "UPDATE uranus.user_organization_link SET permissions=0 WHERE user_uuid=$1", socialOtherUser)
	assertDenied()
	dbExec(t, h, "UPDATE uranus.user_organization_link SET permissions=$1 WHERE user_uuid=$2", int64(app.UserPermEditOrg), socialOtherUser)
	assertSocialStatus(t, socialRequest(r, "GET", path, otherToken, ""), 200)
	if len(socialListData(t, socialRequest(r, "GET", socialAccountsPath, otherToken, ""))) != 2 {
		t.Fatal("authorized list must exclude the other organization")
	}
	// Transfer requires permission in both the current and destination org.
	assertSocialStatus(t, socialRequest(r, "PUT", path, otherToken, `{"org_uuid":"`+socialOtherOrg+`"}`), 403)
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"org_uuid":"`+socialOtherOrg+`","remote_account_id":"43"}`), 200)
	assertSocialStatus(t, socialRequest(r, "GET", path, otherToken, ""), 403)
	storedSocialTokens(t, h, id, "secret-access", "secret-refresh")
	// Unknown organizations have no grant and are rejected without exposing existence.
	assertSocialStatus(t, socialRequest(r, "POST", socialAccountsPath, token, socialBody("facebook", socialOtherUser, "99")), 403)
}

func TestSocialAccountsPostgresMigration(t *testing.T) {
	h, r, token := socialDatabase(t)
	insert := "INSERT INTO " + h.DbSchema + ".social_account (uuid,org_uuid,platform,name,remote_account_id) VALUES ($1,$2,$3,$4,$5)"
	for _, args := range [][]any{
		{authTestUser, socialOrg, "invalid", "name", "id"},
		{authTestUser, socialOtherUser, "instagram", "name", "id"},
		{authTestUser, nil, "instagram", "name", "id"},
		{authTestUser, socialOrg, "instagram", "name", ""},
		{authTestUser, socialOrg, "instagram", "", "id"},
	} {
		if _, err := h.DbPool.Exec(context.Background(), insert, args...); err == nil {
			t.Fatal("migration accepted invalid row")
		}
	}
	w := socialRequest(r, "POST", socialAccountsPath, token, socialBody("instagram", socialOrg, "42"))
	assertSocialStatus(t, w, 201)
	id := socialAccountData(t, w).Uuid
	if _, err := h.DbPool.Exec(context.Background(), insert, authTestUser, socialOrg, "instagram", "name", "42"); err == nil {
		t.Fatal("migration accepted duplicate identity")
	}
	dbExec(t, h, "DELETE FROM uranus.organization WHERE uuid=$1", socialOrg)
	assertSocialStatus(t, socialRequest(r, "GET", socialAccountsPath+"/"+id, token, ""), 404)
	applySocialMigration(t, h, "down")
	applySocialMigration(t, h, "up")
	if len(socialListData(t, socialRequest(r, "GET", socialAccountsPath, token, ""))) != 0 {
		t.Fatal("migration roundtrip failed")
	}
}

func captureSocialLogs(t *testing.T) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "logs")
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, logger := os.Stdout, os.Stderr, log.Writer()
	writer, errorWriter := gin.DefaultWriter, gin.DefaultErrorWriter
	os.Stdout, os.Stderr = file, file
	log.SetOutput(file)
	gin.DefaultWriter, gin.DefaultErrorWriter = file, file
	t.Cleanup(func() {
		os.Stdout, os.Stderr = stdout, stderr
		log.SetOutput(logger)
		gin.DefaultWriter, gin.DefaultErrorWriter = writer, errorWriter
		if err := file.Close(); err != nil {
			t.Error(err)
		}
		data, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "secret-access") || strings.Contains(string(data), "secret-refresh") {
			t.Error("logs exposed credentials")
		}
	})
}

func TestSocialAccountsPostgresRollbackAndSecretErrors(t *testing.T) {
	h, r, token := socialDatabase(t)
	captureSocialLogs(t)
	w := socialRequest(r, "POST", socialAccountsPath, token, socialBody("instagram", socialOrg, "42"))
	assertSocialStatus(t, w, 201)
	id := socialAccountData(t, w).Uuid
	path := socialAccountsPath + "/" + id
	assertSocialStatus(t, socialRequest(r, "POST", socialAccountsPath, token, socialBody("instagram", socialOrg, "42")), 409)
	// A failing-row DETAIL includes both tokens on a check violation.
	dbExec(t, h, "ALTER TABLE uranus.social_account ADD CONSTRAINT reject_name CHECK (name <> 'reject')")
	assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"name":"reject","access_token":"secret-access-new"}`), 400)
	storedSocialTokens(t, h, id, "secret-access", "secret-refresh")
	dbExec(t, h, "ALTER TABLE uranus.social_account DROP CONSTRAINT reject_name")
	// Exercise both statement and commit failures with secret-bearing messages.
	for _, deferred := range []bool{false, true} {
		dbExec(t, h, `CREATE FUNCTION uranus.fail_social_write() RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN RAISE EXCEPTION 'secret-access secret-refresh'; END $$`)
		trigger := "CREATE TRIGGER fail_social AFTER INSERT OR UPDATE OR DELETE ON uranus.social_account"
		if deferred {
			trigger = "CREATE CONSTRAINT TRIGGER fail_social AFTER INSERT OR UPDATE OR DELETE ON uranus.social_account DEFERRABLE INITIALLY DEFERRED"
		}
		dbExec(t, h, trigger+" FOR EACH ROW EXECUTE FUNCTION uranus.fail_social_write()")
		assertSocialStatus(t, socialRequest(r, "POST", socialAccountsPath, token, socialBody("bluesky", socialOrg, "new")), 500)
		assertSocialStatus(t, socialRequest(r, "PUT", path, token, `{"name":"changed","access_token":"secret-access-new","refresh_token":"secret-refresh-new"}`), 500)
		assertSocialStatus(t, socialRequest(r, "DELETE", path, token, ""), 500)
		storedSocialTokens(t, h, id, "secret-access", "secret-refresh")
		w = socialRequest(r, "GET", path, token, "")
		assertSocialStatus(t, w, 200)
		if socialAccountData(t, w).Name != "@test" || len(socialListData(t, socialRequest(r, "GET", socialAccountsPath, token, ""))) != 1 {
			t.Fatal("failed write was not rolled back")
		}
		dbExec(t, h, "DROP TRIGGER fail_social ON uranus.social_account")
		dbExec(t, h, "DROP FUNCTION uranus.fail_social_write()")
	}
}
