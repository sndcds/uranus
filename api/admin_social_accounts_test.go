package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sndcds/uranus/app"
)

const socialAccountsPath = "/api/admin/social/accounts"

func socialRouter(h *ApiHandler) *gin.Engine {
	r := gin.New()
	h.RegisterSocialAccountRoutes(r.Group("/api/admin", app.JWTMiddleware))
	return r
}

func socialToken(t *testing.T, user string) string {
	t.Helper()
	return signedTestToken(t, &app.Claims{
		UserUuid: user, TokenType: app.AccessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}, jwt.SigningMethodHS256, false)
}

func socialRequest(r http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func assertSocialStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status = %d, want %d: %s", w.Code, want, w.Body)
	}
	for _, secret := range []string{`"access_token"`, `"refresh_token"`, "secret-access", "secret-refresh"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatal("response exposed credentials")
		}
	}
}

func TestSocialAccountsRequireAuthentication(t *testing.T) {
	h := setupAuthTest(t)
	r := socialRouter(h)
	for _, route := range []struct{ method, path string }{
		{"GET", socialAccountsPath}, {"POST", socialAccountsPath},
		{"GET", socialAccountsPath + "/" + authTestUser},
		{"PUT", socialAccountsPath + "/" + authTestUser},
		{"DELETE", socialAccountsPath + "/" + authTestUser},
	} {
		for _, token := range []string{"", "invalid-token"} {
			assertSocialStatus(t, socialRequest(r, route.method, route.path, token, "{}"), 401)
		}
	}
	// Cookie mutations use the same origin protection as all other admin routes.
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		path := socialAccountsPath
		if method != "POST" {
			path += "/" + authTestUser
		}
		req := httptest.NewRequest(method, path, strings.NewReader("{}"))
		req.AddCookie(&http.Cookie{Name: "access_token", Value: socialToken(t, authTestUser)})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assertSocialStatus(t, w, 403)
	}
}

func TestSocialAccountsValidation(t *testing.T) {
	h := setupAuthTest(t) // nil database proves validation runs before database access.
	r, token := socialRouter(h), socialToken(t, authTestUser)
	for _, body := range []string{
		"null", "[]", "{", "{}",
		`{"org_uuid":"invalid","platform":"instagram","name":"test","remote_account_id":"42"}`,
		`{"org_uuid":"` + authTestUser + `","platform":"invalid","name":"test","remote_account_id":"42"}`,
		`{"platform":"instagram","name":"test","remote_account_id":"42"}`,
		`{"org_uuid":"` + authTestUser + `","platform":"instagram","name":"test"}`,
		`{"org_uuid":"` + authTestUser + `","platform":"instagram","name":"test","remote_account_id":"  "}`,
		`{"org_uuid":"` + authTestUser + `","platform":"instagram","name":" ","remote_account_id":"42"}`,
		`{"org_uuid":"` + authTestUser + `","platform":"instagram","name":"test","remote_account_id":"42","enabled":null}`,
		`{"access_token":123,"refresh_token":"secret-refresh"}`,
		`{"token_expires_at":"secret-access"}`,
		`{"secret-access":"unknown field"}`,
		`{} {"access_token":"secret-access"}`,
	} {
		assertSocialStatus(t, socialRequest(r, "POST", socialAccountsPath, token, body), 400)
	}
	for _, body := range []string{
		"null", `{"org_uuid":null}`, `{"platform":null}`, `{"name":null}`,
		`{"remote_account_id":""}`, `{"enabled":null}`, `{"platform":"meta"}`,
		`{"org_uuid":"bad"}`, `{"refresh_token":false}`,
	} {
		assertSocialStatus(t, socialRequest(r, "PUT", socialAccountsPath+"/"+authTestUser, token, body), 400)
	}
	for _, method := range []string{"GET", "PUT", "DELETE"} {
		assertSocialStatus(t, socialRequest(r, method, socialAccountsPath+"/invalid", token, "{}"), 400)
	}
	assertSocialStatus(t, socialRequest(r, "GET", socialAccountsPath+"?org_uuid=invalid", token, ""), 400)
}

func TestSocialAccountCredentialsAreWriteOnly(t *testing.T) {
	var input socialAccountInput
	if err := json.Unmarshal([]byte(`{"access_token":"secret-access","refresh_token":"secret-refresh"}`), &input); err != nil {
		t.Fatal(err)
	}
	if !input.AccessToken.Set || input.AccessToken.Value == nil || *input.AccessToken.Value != "secret-access" ||
		!input.RefreshToken.Set || input.RefreshToken.Value == nil || *input.RefreshToken.Value != "secret-refresh" {
		t.Fatal("credentials were not decoded")
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{string(data), fmt.Sprintf("%v %+v %#v", input, input, input)} {
		if strings.Contains(output, "secret-access") || strings.Contains(output, "secret-refresh") {
			t.Fatal("request serialization or formatting exposed a secret")
		}
	}
	for _, code := range []string{"23505", "23503", "23514", "22021", "P0001"} {
		err := socialAccountDBError(&pgconn.PgError{
			Code: code, Message: "secret-access", Detail: "secret-refresh",
		})
		if strings.Contains(fmt.Sprintf("%+v %#v", err, err), "secret-") {
			t.Fatal("database error retained credentials")
		}
	}
}
