package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireAuthOriginDevelopmentDashboard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, origin, referer string
		allowed               bool
	}{
		{"dev dashboard", "https://app.dev.kulturbytes.de", "", true},
		{"prod dashboard", "https://app.kulturbytes.de", "", true},
		{"local development", "http://localhost:5173", "", true},
		{"dev referer", "", "https://app.dev.kulturbytes.de/admin/events", true},
		{"missing provenance", "", "", false},
		{"foreign origin", "https://example.org", "", false},
		{"http dev dashboard", "http://app.dev.kulturbytes.de", "", false},
		{"suffix attack", "https://app.dev.kulturbytes.de.example.org", "", false},
		{"different port", "https://app.dev.kulturbytes.de:444", "", false},
		{"foreign origin with trusted referer", "https://example.org", "https://app.dev.kulturbytes.de/", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/admin/refresh", nil)
			if tc.origin != "" {
				ctx.Request.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				ctx.Request.Header.Set("Referer", tc.referer)
			}
			if got := RequireAuthOrigin(ctx); got != tc.allowed {
				t.Fatalf("allowed = %v, want %v", got, tc.allowed)
			}
			if !tc.allowed && recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", recorder.Code)
			}
		})
	}
}
