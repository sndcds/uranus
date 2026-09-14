package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sndcds/uranus/app"
)

const authTestUser = "01994126-6680-7000-8000-000000000001"

func setupAuthTest(t *testing.T) *ApiHandler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	old := app.UranusInstance
	app.UranusInstance = &app.Uranus{JwtKey: []byte("auth-unit-test-key-not-for-production")}
	t.Cleanup(func() { app.UranusInstance = old })
	config := app.DefaultConfig()
	return &ApiHandler{Config: &config}
}

func signedTestToken(t *testing.T, claims *app.Claims, method jwt.SigningMethod, wrongKey bool) string {
	t.Helper()
	key := app.UranusInstance.JwtKey
	if wrongKey {
		key = []byte("wrong-key")
	}
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func authRequest(r http.Handler, path, token string, cookie bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	if cookie {
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: token})
		req.Header.Set("Origin", "https://app.kulturbytes.de")
	} else if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRejectInvalidRefreshBeforeDatabase(t *testing.T) {
	h := setupAuthTest(t)
	r := gin.New()
	h.RegisterSessionRoutes(r)
	for _, tc := range []struct {
		name, typ, user, jti                    string
		expired, noExp, badKey, missing, future bool
		method                                  jwt.SigningMethod
	}{
		{name: "missing", missing: true},
		{name: "signature", typ: "refresh", user: authTestUser, jti: authTestUser, badKey: true},
		{name: "algorithm", typ: "refresh", user: authTestUser, jti: authTestUser, method: jwt.SigningMethodHS512},
		{name: "access", typ: "access", user: authTestUser, jti: authTestUser},
		{name: "expired", typ: "refresh", user: authTestUser, jti: authTestUser, expired: true},
		{name: "missing jti", typ: "refresh", user: authTestUser},
		{name: "malformed jti", typ: "refresh", user: authTestUser, jti: "bad"},
		{name: "missing user", typ: "refresh", jti: authTestUser},
		{name: "malformed user", typ: "refresh", user: "bad", jti: authTestUser},
		{name: "missing exp", typ: "refresh", user: authTestUser, jti: authTestUser, noExp: true},
		{name: "future nbf", typ: "refresh", user: authTestUser, jti: authTestUser, future: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &app.Claims{UserUuid: tc.user, TokenType: tc.typ, RegisteredClaims: jwt.RegisteredClaims{ID: tc.jti, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
			if tc.expired {
				c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
			}
			if tc.noExp {
				c.ExpiresAt = nil
			}
			if tc.future {
				c.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour))
			}
			method := tc.method
			if method == nil {
				method = jwt.SigningMethodHS256
			}
			token := signedTestToken(t, c, method, tc.badKey)
			if tc.missing {
				token = ""
			}
			for _, cookie := range []bool{false, true} {
				w := authRequest(r, "/api/admin/refresh", token, cookie)
				if w.Code != 401 {
					t.Fatalf("got %d: %s", w.Code, w.Body)
				}
				if len(w.Result().Cookies()) != 0 {
					t.Fatal("failed refresh must not issue cookies")
				}
			}
		})
	}
}

func TestRefreshCSRFBeforeDatabase(t *testing.T) {
	h := setupAuthTest(t)
	r := gin.New()
	h.RegisterSessionRoutes(r)
	for _, path := range []string{"/api/admin/refresh", "/api/admin/logout"} {
		for _, origin := range []string{"", "https://evil.kulturbytes.de", "null"} {
			req := httptest.NewRequest("POST", path, nil)
			req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "anything"})
			req.Header.Set("Origin", origin)
			// A header must not bypass CSRF when the cookie supplies authentication.
			req.Header.Set("Authorization", "Bearer anything")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != 403 || len(w.Result().Cookies()) != 0 {
				t.Fatalf("unexpected CSRF response: %d", w.Code)
			}
		}
	}
}

func TestAccessUUIDHelperRejectsRefresh(t *testing.T) {
	h := setupAuthTest(t)
	pair, err := h.newAuthTokenPair(authTestUser)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ token, want string }{{pair.accessToken, authTestUser}, {pair.refreshToken, ""}, {"invalid", ""}} {
		gc, _ := gin.CreateTestContext(httptest.NewRecorder())
		gc.Request = httptest.NewRequest("GET", "/", nil)
		gc.Request.Header.Set("Authorization", "Bearer "+tc.token)
		if got := UseUuidFromAccessToken(gc); got != tc.want {
			t.Fatalf("unexpected user %s", got)
		}
	}
}

func TestTokenLifetimesAndCookies(t *testing.T) {
	h := setupAuthTest(t)
	h.Config.AuthTokenExpirationTime = 3600
	pair, err := h.newAuthTokenPair(authTestUser)
	if err != nil {
		t.Fatal(err)
	}
	if pair.accessClaims.ExpiresAt.Sub(pair.accessClaims.IssuedAt.Time) != 15*time.Minute {
		t.Fatal("access lifetime not bounded")
	}
	if pair.refreshClaims.ID == "" || pair.accessClaims.ID != "" {
		t.Fatal("unexpected token IDs")
	}
	for _, dev := range []bool{false, true} {
		h.Config.DevMode = dev
		w := httptest.NewRecorder()
		gc, _ := gin.CreateTestContext(w)
		h.setAuthCookies(gc, pair)
		cookies := w.Result().Cookies()
		if len(cookies) != 2 {
			t.Fatal("missing cookies")
		}
		for _, c := range cookies {
			if !c.HttpOnly || c.Secure == dev || c.Domain != "" || c.SameSite != http.SameSiteLaxMode || c.MaxAge <= 0 || !c.Expires.After(time.Now()) {
				t.Fatalf("unsafe cookie: %+v", c)
			}
			if c.Name == "access_token" && c.Path != "/api" || c.Name == "refresh_token" && c.Path != "/api/admin" {
				t.Fatal("wrong cookie path")
			}
		}
	}
	h.Config.RefreshTokenExpirationTime = 1
	if _, err := h.newAuthTokenPair(authTestUser); err == nil {
		t.Fatal("invalid lifetimes accepted")
	}
}

func TestLogoutMissingTokenClearsCookies(t *testing.T) {
	h := setupAuthTest(t)
	r := gin.New()
	h.RegisterSessionRoutes(r)
	w := authRequest(r, "/api/admin/logout", "", false)
	if w.Code != 401 || len(w.Result().Cookies()) != 2 {
		t.Fatalf("unexpected missing-token logout: %d", w.Code)
	}
	for _, c := range w.Result().Cookies() {
		if c.Value != "" || c.MaxAge != -1 {
			t.Fatal("cookie not deleted")
		}
	}
}

func TestTokenIssuanceRequiresKey(t *testing.T) {
	h := setupAuthTest(t)
	app.UranusInstance.JwtKey = nil
	if _, err := h.newAuthTokenPair(authTestUser); err == nil {
		t.Fatal("issued tokens without signing key")
	}
}
