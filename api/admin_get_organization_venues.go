package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: User must be authenticated.
// Only returns venues for the organization if the authenticated user is linked via `user_organization_link`.
// If the user is not linked, returns HTTP 403 Forbidden.
// PermissionChecks: Already enforced in SQL.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetOrganizationVenues(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-organization-venues")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "invalid orgUuid")
		return
	}

	var err error

	// Start date for getting upcoming event counts
	startStr := gc.Query("start")
	var startDate time.Time
	if startStr != "" {
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			startDate = time.Now() // fallback on parse error
		}
	} else {
		startDate = time.Now() // fallback if param missing
	}

	row := h.DbPool.QueryRow(ctx, app.UranusInstance.SqlAdminGetOrganizationVenues, userUuid, orgUuid, startDate)
	var jsonResult []byte
	err = row.Scan(&jsonResult)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apiRequest.Error(http.StatusForbidden, "user not linked to organization")
			return
		}
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	if len(jsonResult) == 0 || string(jsonResult) == "[]" {
		apiRequest.Error(http.StatusForbidden, "user not linked to organization")
		return
	}

	// The SQL currently returns an array like: [{...}], so we unmarshal and return first element
	var organizations []map[string]interface{}
	if err := json.Unmarshal(jsonResult, &organizations); err != nil {
		apiRequest.InvalidJSONInput()
		return
	}

	if len(organizations) == 0 {
		apiRequest.Error(http.StatusForbidden, "user not linked to organization")
		return
	}

	apiRequest.Success(http.StatusOK, organizations[0], "")
}
