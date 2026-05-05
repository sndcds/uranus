package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: User must be authenticated.
// The endpoint returns space details only if the authenticated user
// is linked to the space (via the SQL query).
// PermissionChecks: Enforced in SQL; no additional checks needed in Go.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetSpace(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-space")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	spaceUuid := gc.Param("spaceUuid")
	if spaceUuid == "" {
		apiRequest.Required("spaceUuid is required")
		return
	}
	apiRequest.SetMeta("space_uuid", spaceUuid)

	var space model.Space
	query := app.UranusInstance.SqlAdminGetSpace
	row := h.DbPool.QueryRow(ctx, query, spaceUuid, userUuid)

	err := row.Scan(
		&space.Uuid,
		&space.Name,
		&space.Description,
		&space.SpaceType,
		&space.BuildingLevel,
		&space.TotalCapacity,
		&space.SeatingCapacity,
		&space.WebLink,
		&space.AccessibilityFlags,
		&space.AccessibilitySummary,
		&space.AreaSqm,
	)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, space, "space loaded successfully")
}
