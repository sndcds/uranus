package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetPortalGeoJSON(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-portal-geojson")
	ctx := gc.Request.Context()

	apiRequest.SetMeta("api_url", h.Config.BaseApiUrl)

	portalUuid := gc.Param("uuid")

	if portalUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "portal uuid is required")
		return
	}

	apiRequest.SetMeta("portal", portalUuid)

	query := app.UranusInstance.SqlGetPortalGeoJSON

	var geojson map[string]interface{}

	err := h.DbPool.QueryRow(
		ctx,
		query,
		portalUuid,
	).Scan(&geojson)

	if err != nil {
		if err == pgx.ErrNoRows {
			apiRequest.Error(http.StatusNotFound, "portal not found")
			return
		}

		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, geojson)
}
