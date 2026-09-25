package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func contentContext(userUuid string) *gin.Context {
	gc, _ := gin.CreateTestContext(httptest.NewRecorder())
	gc.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	gc.Set("user-uuid", userUuid)
	return gc
}

func TestContentItemValidation(t *testing.T) {
	h := setupAuthTest(t) // A nil pool proves rejection happens before database access.
	for _, input := range []struct {
		name, user, org, sourceType, source string
		status                              int
	}{
		{"unknown type", authTestUser, socialOrg, "post", authTestUser, 400},
		{"missing type", authTestUser, socialOrg, "", authTestUser, 400},
		{"invalid source", authTestUser, socialOrg, "event", "invalid", 400},
		{"nil source", authTestUser, socialOrg, "event", "00000000-0000-0000-0000-000000000000", 400},
		{"missing source", authTestUser, socialOrg, "venue", "", 400},
		{"invalid organization", authTestUser, "invalid", "organization", socialOrg, 400},
		{"missing authentication", "", socialOrg, "event", authTestUser, 401},
	} {
		t.Run(input.name, func(t *testing.T) {
			item, txErr := h.LoadContentItem(contentContext(input.user), input.org, input.sourceType, input.source, "de")
			if item != nil || txErr == nil || txErr.Code != input.status {
				t.Fatalf("item=%+v error=%+v", item, txErr)
			}
		})
	}

	for _, input := range []struct {
		user, post string
		status     int
	}{
		{authTestUser, "invalid", 400}, {authTestUser, "", 400}, {"", authTestUser, 401},
	} {
		item, txErr := h.LoadSocialPostContentItem(contentContext(input.user), input.post, "de")
		if item != nil || txErr == nil || txErr.Code != input.status {
			t.Fatalf("item=%+v error=%+v", item, txErr)
		}
	}
}

func TestContentItemErrors(t *testing.T) {
	if txErr := contentItemDBError(pgx.ErrNoRows); txErr.Code != http.StatusNotFound {
		t.Fatalf("unexpected missing-source error: %+v", txErr)
	}
	for _, txErr := range []*ApiTxError{
		contentItemDBError(errors.New("secret-access secret-refresh")),
		contentItemError(&ApiTxError{Code: 500, Err: errors.New("secret-access secret-refresh")}),
	} {
		if txErr.Err != nil || txErr.Code != 500 || txErr.Message != "internal server error" {
			t.Fatalf("database details retained: %+v", txErr)
		}
	}
}
