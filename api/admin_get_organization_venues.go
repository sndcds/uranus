package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: User must be authenticated.
// Only returns venues for the organization if the authenticated user is linked via `user_organization_link`.
// If the user is not linked, returns HTTP 403 Forbidden.
// PermissionChecks: Already enforced in SQL.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetOrganizationVenues(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organizationId"})
		return
	}

	var err error
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

	// Run query
	row := h.DbPool.QueryRow(ctx, app.UranusInstance.SqlAdminGetOrganizationVenues, userId, organizationId, startDate)

	var jsonResult []byte
	if err := row.Scan(&jsonResult); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusForbidden, gin.H{"error": "user not linked to organization"})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(jsonResult) == 0 || string(jsonResult) == "[]" {
		gc.JSON(http.StatusForbidden, gin.H{"error": "user not linked to organization"})
		return
	}

	// The SQL currently returns an array like: [{...}], so we unmarshal and return first element
	var organizations []map[string]interface{}
	if err := json.Unmarshal(jsonResult, &organizations); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse JSON"})
		return
	}

	if len(organizations) == 0 {
		gc.JSON(http.StatusForbidden, gin.H{"error": "user not linked to organization"})
		return
	}

	// Return only the first organization as an object
	singleOrganizationJSON, err := json.Marshal(organizations[0])
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode JSON"})
		return
	}

	gc.Data(http.StatusOK, "application/json", singleOrganizationJSON)
}
