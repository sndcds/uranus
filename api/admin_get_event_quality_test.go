package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func TestAdminGetEventQualityInvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &ApiHandler{} // Invalid input must be rejected before touching the database.
	for _, id := range []string{"", "invalid", "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"} {
		recorder := httptest.NewRecorder()
		gc, _ := gin.CreateTestContext(recorder)
		gc.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		gc.Params = gin.Params{{Key: "eventUuid", Value: id}}
		h.AdminGetEventQuality(gc)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%q: got %d", id, recorder.Code)
		}
		var response struct {
			ResponseType string `json:"response_type"`
			Status       int    `json:"status"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Status != 400 || response.ResponseType != "admin-get-event-quality" {
			t.Fatalf("unexpected response: %+v", response)
		}
	}
}

func TestAdminGetEventQualityRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api/admin", app.JWTMiddleware)
	h := &ApiHandler{}
	admin.GET("/event/:eventUuid/quality", h.AdminGetEventQuality)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/event/0198b7c0-0000-7000-8000-000000000001/quality", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("got %d: %s", recorder.Code, recorder.Body.String())
	}
}
