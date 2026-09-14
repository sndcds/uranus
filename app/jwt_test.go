package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testUserUUID = "01994126-6680-7000-8000-000000000001"

func TestJWTMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := UranusInstance
	UranusInstance = &Uranus{JwtKey: []byte("test-signing-key-with-enough-entropy")}
	t.Cleanup(func() { UranusInstance = old })
	for _, tc := range []struct {
		name, typ, user                                 string
		method                                          jwt.SigningMethod
		expired, missing, badKey, cookie, noExp, future bool
		want                                            int
	}{
		{name: "valid access", typ: AccessTokenType, user: testUserUUID, want: 200},
		{name: "cookie fallback", typ: AccessTokenType, user: testUserUUID, cookie: true, want: 200},
		{name: "missing", missing: true, want: 401},
		{name: "signature", typ: AccessTokenType, user: testUserUUID, badKey: true, want: 401},
		{name: "algorithm", typ: AccessTokenType, user: testUserUUID, method: jwt.SigningMethodHS384, want: 401},
		{name: "expired", typ: AccessTokenType, user: testUserUUID, expired: true, want: 401},
		{name: "refresh", typ: RefreshTokenType, user: testUserUUID, want: 401},
		{name: "missing user", typ: AccessTokenType, want: 401},
		{name: "malformed user", typ: AccessTokenType, user: "broken", want: 401},
		{name: "nil user", typ: AccessTokenType, user: "00000000-0000-0000-0000-000000000000", want: 401},
		{name: "missing type", user: testUserUUID, want: 401},
		{name: "missing expiration", typ: AccessTokenType, user: testUserUUID, noExp: true, want: 401},
		{name: "not yet valid", typ: AccessTokenType, user: testUserUUID, future: true, want: 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := &Claims{UserUuid: tc.user, TokenType: tc.typ, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
			if tc.expired {
				claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
			}
			if tc.noExp {
				claims.ExpiresAt = nil
			}
			if tc.future {
				claims.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour))
			}
			method := tc.method
			if method == nil {
				method = jwt.SigningMethodHS256
			}
			key := UranusInstance.JwtKey
			if tc.badKey {
				key = []byte("wrong-key")
			}
			token, err := jwt.NewWithClaims(method, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			r := gin.New()
			r.GET("/", JWTMiddleware, func(c *gin.Context) {
				if c.GetString("user-uuid") != testUserUUID {
					t.Error("missing user context")
				}
				if c.MustGet("jwt-claims").(*Claims).TokenType != AccessTokenType {
					t.Error("missing claims")
				}
				c.Status(200)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if !tc.missing {
				if tc.cookie {
					req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
				} else {
					req.Header.Set("Authorization", "Bearer "+token)
				}
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("got %d: %s", w.Code, w.Body)
			}
		})
	}
	// The shared parser is intentionally purpose agnostic.
	refresh := &Claims{TokenType: RefreshTokenType, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, refresh).SignedString(UranusInstance.JwtKey)
	if _, err := ParseJWT(token); err != nil {
		t.Fatal(err)
	}
}

func TestCookieCSRFAndBearerPrecedence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := UranusInstance
	UranusInstance = &Uranus{JwtKey: []byte("test-key")}
	t.Cleanup(func() { UranusInstance = old })
	claims := &Claims{UserUuid: testUserUUID, TokenType: AccessTokenType, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(UranusInstance.JwtKey)
	r := gin.New()
	r.POST("/", JWTMiddleware, func(c *gin.Context) { c.Status(200) })
	for _, tc := range []struct {
		name, origin, referer, bearer string
		want                          int
	}{
		{name: "trusted", origin: "https://app.kulturbytes.de", want: 200},
		{name: "localhost", origin: "http://localhost:5173", want: 200},
		{name: "missing provenance", want: 403},
		{name: "hostile sibling", origin: "https://evil.kulturbytes.de", want: 403},
		{name: "null origin", origin: "null", want: 403},
		{name: "referer fallback", referer: "https://app.kulturbytes.de/events", want: 200},
		{name: "bad origin overrides referer", origin: "null", referer: "https://app.kulturbytes.de/", want: 403},
		{name: "bearer without provenance", bearer: token, want: 200},
		{name: "invalid bearer overrides cookie", bearer: "invalid", want: 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Referer", tc.referer)
			if tc.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+tc.bearer)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("got %d", w.Code)
			}
		})
	}
}

func TestJWTRejectsUnconfiguredKey(t *testing.T) {
	old := UranusInstance
	t.Cleanup(func() { UranusInstance = old })
	claims := &Claims{UserUuid: testUserUUID, TokenType: AccessTokenType, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	for _, instance := range []*Uranus{nil, {}} {
		UranusInstance = instance
		if _, err := ParseJWT(token); err == nil {
			t.Fatal("empty-key token accepted")
		}
	}
	UranusInstance = &Uranus{JwtKey: []byte("configured-secret")}
	if _, err := ParseJWT(token); err == nil {
		t.Fatal("legacy empty-key token accepted")
	}
}
