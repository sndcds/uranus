package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sndcds/uranus/app"
)

const socialPostsPath = "/api/admin/social/posts"

func socialPostRouter(h *ApiHandler) *gin.Engine {
	r := gin.New()
	h.RegisterSocialPostRoutes(r.Group("/api/admin", app.JWTMiddleware))
	return r
}

func TestSocialPostsRequireAuthentication(t *testing.T) {
	h := setupAuthTest(t)
	r := socialPostRouter(h)
	for _, route := range []struct{ method, path string }{
		{"GET", socialPostsPath}, {"POST", socialPostsPath},
		{"GET", socialPostsPath + "/" + authTestUser},
		{"PUT", socialPostsPath + "/" + authTestUser},
		{"DELETE", socialPostsPath + "/" + authTestUser},
	} {
		for _, token := range []string{"", "invalid-token"} {
			assertSocialStatus(t, socialRequest(r, route.method, route.path, token, "{}"), 401)
		}
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		path := socialPostsPath
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

func TestSocialPostsValidation(t *testing.T) {
	h := setupAuthTest(t)
	r, token := socialPostRouter(h), socialToken(t, authTestUser)
	for _, body := range []string{"null", "[]", "{", "{}", `{"source_type":"event"}`} {
		assertSocialStatus(t, socialRequest(r, "POST", socialPostsPath, token, body), 400)
	}
	for _, field := range []string{
		`"org_uuid":null`, `"org_uuid":"bad"`, `"source_type":null`,
		`"source_type":"template"`, `"source_type":""`, `"source_uuid":null`, `"source_uuid":"bad"`,
		`"targets":null`, `"targets":{}`, `"targets":[null]`, `"targets":[{}]`,
		`"targets":[{"social_account_uuid":"invalid"}]`,
		`"targets":[{"social_account_uuid":"` + authTestUser + `"},{"social_account_uuid":"` + authTestUser + `"}]`,
		`"targets":[{"social_account_uuid":"` + authTestUser + `","status":"published"}]`,
		`"created_by":"` + authTestUser + `"`, `"access_token":"secret-access"`,
		`"targets":[{"refresh_token":"secret-refresh"}]`,
	} {
		for _, method := range []string{"POST", "PUT"} {
			path := socialPostsPath
			if method == "PUT" {
				path += "/" + authTestUser
			}
			assertSocialStatus(t, socialRequest(r, method, path, token, "{"+field+"}"), 400)
		}
	}
	for _, body := range []string{"null", "[]", "{", `{} {"access_token":"secret-access"}`} {
		assertSocialStatus(t, socialRequest(r, "PUT", socialPostsPath+"/"+authTestUser, token, body), 400)
	}
	for _, method := range []string{"GET", "PUT", "DELETE"} {
		assertSocialStatus(t, socialRequest(r, method, socialPostsPath+"/invalid", token, "{}"), 400)
	}
	assertSocialStatus(t, socialRequest(r, "GET", socialPostsPath+"?org_uuid=invalid", token, ""), 400)

	// UUID spelling must not bypass duplicate detection.
	var payload socialPostInput
	body := `{"targets":[{"social_account_uuid":"01994126-6680-7000-8000-abcdef123456"},{"social_account_uuid":"01994126-6680-7000-8000-ABCDEF123456"}]}`
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.validate(false) != "duplicate social_account_uuid" {
		t.Fatal("equivalent UUIDs were not rejected")
	}
}

func TestSocialPostErrorsDoNotExposeSecrets(t *testing.T) {
	for _, code := range []string{"23505", "23503", "23514", "22021", "P0001"} {
		err := socialPostDBError(&pgconn.PgError{Code: code, Message: "secret-access", Detail: "secret-refresh"})
		if strings.Contains(fmt.Sprintf("%+v %#v", err, err), "secret-") {
			t.Fatal("database error retained credentials")
		}
	}
}
