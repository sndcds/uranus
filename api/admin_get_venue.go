package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: User must be authenticated.
// The endpoint returns venue details only if the authenticated user
// is linked to the venue (via the SQL query).
// PermissionChecks: Already enforced in SQL; no additional checks needed in Go.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	venueId := gc.Param("venueId")
	if venueId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "venue Id is required"})
		return
	}

	query := app.UranusInstance.SqlGetAdminVenue
	rows, err := h.DbPool.Query(ctx, query, venueId, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	if !rows.Next() {
		gc.JSON(http.StatusNotFound, gin.H{"error": "venue not found"})
		return
	}

	fieldDescriptions := rows.FieldDescriptions()
	values, err := rows.Values()
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]interface{}, len(values))
	for i, fd := range fieldDescriptions {
		result[string(fd.Name)] = values[i]
	}

	gc.JSON(http.StatusOK, result)
}
