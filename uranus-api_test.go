package main

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())
	for _, path := range []string{"/api/login", "/api/admin/refresh", "/api/admin/logout"} {
		r.POST(path, func(c *gin.Context) { c.Status(200) })
	}
	for _, path := range []string{"/api/login", "/api/admin/refresh", "/api/admin/logout"} {
		for _, origin := range []string{"https://app.kulturbytes.de", "http://localhost:5173", "https://evil.kulturbytes.de", "null"} {
			for _, method := range []string{"POST", "OPTIONS"} {
				req := httptest.NewRequest(method, path, nil)
				req.Header.Set("Origin", origin)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if origin == "https://app.kulturbytes.de" || origin == "http://localhost:5173" {
					want := 200
					if method == "OPTIONS" {
						want = 204
					}
					if w.Code != want || w.Header().Get("Access-Control-Allow-Origin") != origin || w.Header().Get("Access-Control-Allow-Credentials") != "true" {
						t.Fatalf("credentialed CORS failed for %s %s", method, path)
					}
				} else if w.Code != 403 || w.Header().Get("Access-Control-Allow-Origin") != "" {
					t.Fatalf("untrusted origin accepted for %s", path)
				}
			}
		}
	}
}
