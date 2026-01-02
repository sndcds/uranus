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

// TODO: Review code

func (h *ApiHandler) AdminGetOrganizationVenues(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
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
			gc.JSON(http.StatusNoContent, gin.H{"error": err.Error()})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// The SQL currently returns an array like: [{...}], so we unmarshal and return first element
	var organizations []map[string]interface{}
	if err := json.Unmarshal(jsonResult, &organizations); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse JSON"})
		return
	}

	if len(organizations) == 0 {
		gc.JSON(http.StatusNoContent, gin.H{"error": "organization not found"})
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
