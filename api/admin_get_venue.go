package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
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
	apiRequest := grains_api.NewRequest(gc, "admin-get-venue")

	venueId := gc.Param("venueId")
	if venueId == "" {
		apiRequest.Error(http.StatusBadRequest, "venueId is required")
		return
	}

	query := app.UranusInstance.SqlAdminGetVenue
	rows, err := h.DbPool.Query(ctx, query, venueId, userId)
	if err != nil {
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	if !rows.Next() {
		apiRequest.Error(http.StatusNotFound, "venue not found")
		return
	}

	fieldDescriptions := rows.FieldDescriptions()
	values, err := rows.Values()
	if err != nil {
		apiRequest.DatabaseError()
		return
	}

	result := make(map[string]interface{}, len(values))
	for i, fd := range fieldDescriptions {
		result[string(fd.Name)] = values[i]
	}

	apiRequest.Success(http.StatusOK, result, "venue loaded successfully")
}
