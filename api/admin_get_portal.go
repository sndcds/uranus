package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: User must be authenticated.
// PermissionChecks: Enforced in SQL; no additional checks needed in Go.
// Verified: 2026-05-10, Roald

func (h *ApiHandler) AdminGetPortal(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-portal")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	portalUuid := gc.Param("portalUuid")
	if portalUuid == "" {
		apiRequest.Required("portalUuid is required")
		return
	}
	apiRequest.SetMeta("portal_uuid", portalUuid)

	query := app.UranusInstance.SqlAdminGetPortal
	row := h.DbPool.QueryRow(ctx, query, portalUuid, userUuid)

	var portal model.Portal
	portal.Uuid = portalUuid
	err := row.Scan(
		&portal.OrgUuid,
		&portal.Name,
		&portal.Description,
		&portal.SpatialFilterMode,
		&portal.PreFilter,
		&portal.Geometry,
		&portal.Style,
	)
	if err != nil {
		// debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, portal, "portal loaded successfully")
}
