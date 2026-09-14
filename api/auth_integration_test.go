package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
	"golang.org/x/crypto/bcrypt"
)

// Integration tests only connect to an explicitly supplied test database. Each
// test owns an isolated schema and executes the actual versioned migration.
func authDatabase(t *testing.T) (*ApiHandler, *gin.Engine) {
	t.Helper()
	dsn := os.Getenv("URANUS_AUTH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set URANUS_AUTH_TEST_DATABASE_URL to run PostgreSQL authentication tests")
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
	h.DbPool = pool
	h.DbSchema = "auth_test_" + strings.ReplaceAll(id, "-", "")
	dbExec(t, h, "CREATE SCHEMA "+h.DbSchema)
	t.Cleanup(func() { dbExec(t, h, "DROP SCHEMA "+h.DbSchema+" CASCADE") })
	// Use the actual user table definition, excluding snapshot-only duplicate
	// indexes and the unrelated modified_at trigger.
	ddl, err := os.ReadFile("../ddl/user.ddl")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, strings.Split(string(ddl), "-- Indices")[0])
	applyAuthMigration(t, h, "up")
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, `INSERT INTO uranus."user" (uuid,email,password_hash,is_active) VALUES ($1,$2,$3,true)`, authTestUser, "auth@example.test", string(hash))
	r := gin.New()
	r.POST("/api/login", h.Login)
	h.RegisterSessionRoutes(r)
	r.GET("/api/admin/check", app.JWTMiddleware, func(c *gin.Context) { c.Status(200) })
	return h, r
}

func dbExec(t *testing.T, h *ApiHandler, sql string, args ...any) {
	t.Helper()
	_, err := h.DbPool.Exec(context.Background(), strings.ReplaceAll(sql, "uranus.", h.DbSchema+"."), args...)
	if err != nil {
		t.Fatal(err)
	}
}
func applyAuthMigration(t *testing.T, h *ApiHandler, direction string) {
	t.Helper()
	sql, err := os.ReadFile("../migrations/202609140001_refresh_token." + direction + ".sql")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, string(sql))
}
func loginForTest(t *testing.T, r *gin.Engine) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"email":"auth@example.test","password":"correct-password"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func cookieToken(t *testing.T, w *httptest.ResponseRecorder, name string) string {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name && c.Value != "" {
			return c.Value
		}
	}
	t.Fatalf("missing %s (%d): %s", name, w.Code, w.Body)
	return ""
}
func tokenClaims(t *testing.T, token string) *app.Claims {
	t.Helper()
	c, err := app.ParseJWT(token)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func loginRefreshToken(t *testing.T, r *gin.Engine) string {
	t.Helper()
	w := loginForTest(t, r)
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	return cookieToken(t, w, "refresh_token")
}
func dbCount(t *testing.T, h *ApiHandler, where string, args ...any) int {
	t.Helper()
	var count int
	err := h.DbPool.QueryRow(context.Background(), "SELECT count(*) FROM "+h.DbSchema+".refresh_token WHERE "+where, args...).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func TestAuthPostgresLoginRotationReuse(t *testing.T) {
	h, r := authDatabase(t)
	login := loginForTest(t, r)
	if login.Code != 200 {
		t.Fatalf("login: %d %s", login.Code, login.Body)
	}
	old := cookieToken(t, login, "refresh_token")
	oldClaims := tokenClaims(t, old)
	access := cookieToken(t, login, "access_token")
	if tokenClaims(t, access).TokenType != "access" {
		t.Fatal("wrong access type")
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data["refresh_token"] != old || body.Data["access_token"] != access {
		t.Fatal("login response compatibility lost")
	}
	if dbCount(t, h, "true") != 1 {
		t.Fatal("login must register refresh only")
	}
	otherSession := loginRefreshToken(t, r)
	w := authRequest(r, "/api/admin/refresh", old, true)
	if w.Code != 200 {
		t.Fatalf("refresh: %d %s", w.Code, w.Body)
	}
	next := cookieToken(t, w, "refresh_token")
	nextClaims := tokenClaims(t, next)
	if oldClaims.ID == nextClaims.ID || nextClaims.TokenType != "refresh" {
		t.Fatal("no rotation")
	}
	var refreshed map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &refreshed); err != nil {
		t.Fatal(err)
	}
	if refreshed["refresh_token"] != next || refreshed["access_token"] != cookieToken(t, w, "access_token") {
		t.Fatal("refresh response incompatible")
	}
	if dbCount(t, h, "jti=$1 AND revoked_at IS NOT NULL", oldClaims.ID) != 1 {
		t.Fatal("old token not revoked")
	}
	// A normal Bearer client can rotate the same family again.
	w = authRequest(r, "/api/admin/refresh", next, false)
	if w.Code != 200 {
		t.Fatalf("bearer refresh: %d %s", w.Code, w.Body)
	}
	newest := cookieToken(t, w, "refresh_token")
	w = authRequest(r, "/api/admin/refresh", old, true)
	if w.Code != 401 {
		t.Fatal("reused token accepted")
	}
	if dbCount(t, h, "family_uuid=$1 AND revoked_at IS NULL", oldClaims.ID) != 0 {
		t.Fatal("reuse did not revoke descendants")
	}
	if authRequest(r, "/api/admin/refresh", newest, false).Code != 401 {
		t.Fatal("descendant still usable")
	}
	if authRequest(r, "/api/admin/refresh", otherSession, false).Code != 200 {
		t.Fatal("reuse revoked unrelated session")
	}
	// Issued access tokens remain valid until their short expiration.
	req := httptest.NewRequest("GET", "/api/admin/check", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("access token unexpectedly revoked")
	}
}

func TestAuthPostgresRefreshFailures(t *testing.T) {
	for _, kind := range []string{"unknown jti", "revoked", "database expired", "inactive user", "deleted user", "wrong owner"} {
		t.Run(kind, func(t *testing.T) {
			h, r := authDatabase(t)
			token := loginRefreshToken(t, r)
			c := tokenClaims(t, token)
			switch kind {
			case "unknown jti":
				c.ID, _ = grains_uuid.Uuidv7String()
				token = signedTestToken(t, c, jwt.SigningMethodHS256, false)
			case "revoked":
				dbExec(t, h, "UPDATE uranus.refresh_token SET revoked_at=CURRENT_TIMESTAMP")
			case "database expired":
				dbExec(t, h, "UPDATE uranus.refresh_token SET created_at=CURRENT_TIMESTAMP-interval '2 hours',expires_at=CURRENT_TIMESTAMP-interval '1 hour'")
			case "inactive user":
				dbExec(t, h, `UPDATE uranus."user" SET is_active=false`)
			case "deleted user":
				dbExec(t, h, `DELETE FROM uranus."user"`)
			case "wrong owner":
				c.UserUuid, _ = grains_uuid.Uuidv7String()
				dbExec(t, h, `INSERT INTO uranus."user" (uuid,email,password_hash,is_active) VALUES ($1,'other@example.test','unused',true)`, c.UserUuid)
				token = signedTestToken(t, c, jwt.SigningMethodHS256, false)
			}
			w := authRequest(r, "/api/admin/refresh", token, true)
			if w.Code != 401 {
				t.Fatalf("got %d %s", w.Code, w.Body)
			}
			if len(w.Result().Cookies()) != 0 {
				t.Fatal("failed refresh issued cookies")
			}
		})
	}
}

func TestAuthPostgresLogout(t *testing.T) {
	h, r := authDatabase(t)
	old := loginRefreshToken(t, r)
	w := authRequest(r, "/api/admin/refresh", old, true)
	next := cookieToken(t, w, "refresh_token")
	// Logging out with a rotated token also revokes its current descendant.
	w = authRequest(r, "/api/admin/logout", old, true)
	if w.Code != 200 {
		t.Fatalf("logout: %d %s", w.Code, w.Body)
	}
	if dbCount(t, h, "revoked_at IS NULL") != 0 {
		t.Fatal("logout did not revoke session")
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatal("missing cookie deletion")
	}
	for _, c := range cookies {
		if c.Value != "" || c.MaxAge != -1 || !c.Expires.Before(time.Now()) || !c.HttpOnly || !c.Secure {
			t.Fatal("cookie not cleared safely")
		}
	}
	if authRequest(r, "/api/admin/refresh", next, true).Code != 401 {
		t.Fatal("refresh after logout accepted")
	}
	if authRequest(r, "/api/admin/logout", old, false).Code != 200 {
		t.Fatal("logout is not idempotent")
	}
}

func TestAuthPostgresRotationRollback(t *testing.T) {
	for _, atCommit := range []bool{false, true} {
		t.Run(fmt.Sprintf("commit=%v", atCommit), func(t *testing.T) {
			h, r := authDatabase(t)
			token := loginRefreshToken(t, r)
			c := tokenClaims(t, token)
			if atCommit {
				dbExec(t, h, `CREATE FUNCTION uranus.reject_refresh_insert() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test commit failure'; END $$;
    CREATE CONSTRAINT TRIGGER fail_refresh_commit AFTER INSERT ON uranus.refresh_token DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION uranus.reject_refresh_insert();`)
			} else {
				dbExec(t, h, `ALTER TABLE uranus.refresh_token ADD CONSTRAINT reject_new_tokens CHECK (false) NOT VALID`)
			}
			w := authRequest(r, "/api/admin/refresh", token, true)
			if w.Code != 500 || len(w.Result().Cookies()) != 0 {
				t.Fatalf("expected server failure without cookies: %d %s", w.Code, w.Body)
			}
			if dbCount(t, h, "jti=$1 AND revoked_at IS NULL", c.ID) != 1 || dbCount(t, h, "true") != 1 {
				t.Fatal("rotation did not roll back")
			}
			// Login must also publish no tokens if registration or commit fails.
			w = loginForTest(t, r)
			if w.Code != 500 || len(w.Result().Cookies()) != 0 {
				t.Fatal("failed login published tokens")
			}
			if atCommit {
				dbExec(t, h, `DROP TRIGGER fail_refresh_commit ON uranus.refresh_token`)
			} else {
				dbExec(t, h, `ALTER TABLE uranus.refresh_token DROP CONSTRAINT reject_new_tokens`)
			}
			if authRequest(r, "/api/admin/refresh", token, true).Code != 200 {
				t.Fatal("rollback lost old refresh token")
			}
		})
	}
}

func TestAuthPostgresConcurrentRefresh(t *testing.T) {
	h, r := authDatabase(t)
	token := loginRefreshToken(t, r)
	responses := make(chan *httptest.ResponseRecorder, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; responses <- authRequest(r, "/api/admin/refresh", token, false) }()
	}
	close(start)
	wg.Wait()
	close(responses)
	success, rejected := 0, 0
	for w := range responses {
		switch w.Code {
		case 200:
			success++
		case 401:
			rejected++
		default:
			t.Fatalf("unexpected %d %s", w.Code, w.Body)
		}
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("success=%d rejected=%d", success, rejected)
	}
	if dbCount(t, h, "true") != 2 || dbCount(t, h, "revoked_at IS NULL") != 0 {
		t.Fatal("concurrent reuse left active session")
	}
}

func TestAuthPostgresLogoutRacingRefresh(t *testing.T) {
	h, r := authDatabase(t)
	token := loginRefreshToken(t, r)
	start := make(chan struct{})
	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 2)
	for _, path := range []string{"/api/admin/refresh", "/api/admin/logout"} {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; responses <- authRequest(r, path, token, false) }()
	}
	close(start)
	wg.Wait()
	close(responses)
	for w := range responses {
		if w.Code != 200 && w.Code != 401 {
			t.Fatalf("unexpected status %d", w.Code)
		}
	}
	if dbCount(t, h, "revoked_at IS NULL") != 0 {
		t.Fatal("logout race left active session")
	}
}

func TestAuthPostgresMigration(t *testing.T) {
	h, _ := authDatabase(t)
	id, _ := grains_uuid.Uuidv7String()
	insert := fmt.Sprintf(`INSERT INTO %s.refresh_token (jti,user_uuid,family_uuid,expires_at) VALUES ($1,$2,$1,CURRENT_TIMESTAMP+interval '1 hour')`, h.DbSchema)
	dbExec(t, h, insert, id, authTestUser)
	if _, err := h.DbPool.Exec(context.Background(), insert, id, authTestUser); err == nil {
		t.Fatal("duplicate jti accepted")
	}
	unknown, _ := grains_uuid.Uuidv7String()
	if _, err := h.DbPool.Exec(context.Background(), insert, unknown, unknown); err == nil {
		t.Fatal("missing foreign key")
	}
	dbExec(t, h, `DELETE FROM uranus."user" WHERE uuid=$1`, authTestUser)
	if dbCount(t, h, "true") != 0 {
		t.Fatal("user delete did not cascade")
	}
	applyAuthMigration(t, h, "down")
	applyAuthMigration(t, h, "up")
	if dbCount(t, h, "true") != 0 {
		t.Fatal("migration roundtrip failed")
	}
}

func TestAuthPostgresInvalidLogin(t *testing.T) {
	h, r := authDatabase(t)
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"email":"auth@example.test","password":"wrong-password"}`, 401},
		{`{"email":"unknown@example.test","password":"correct-password"}`, 401},
		{`{"email":"auth@example.test","password":""}`, 400},
	} {
		req := httptest.NewRequest("POST", "/api/login", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.status || len(w.Result().Cookies()) != 0 {
			t.Fatalf("invalid credentials: %d %s", w.Code, w.Body)
		}
	}
	dbExec(t, h, `UPDATE uranus."user" SET is_active=false`)
	if w := loginForTest(t, r); w.Code != 401 || len(w.Result().Cookies()) != 0 {
		t.Fatal("inactive user logged in")
	}
	if dbCount(t, h, "true") != 0 {
		t.Fatal("failed login registered tokens")
	}
}
