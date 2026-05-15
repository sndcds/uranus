package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetPortal(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-portal")
	ctx := gc.Request.Context()

	portalUuid := gc.Param("uuid")
	if portalUuid == "" {
		apiRequest.Required("parameter uuid is required")
		return
	}

	var portal struct {
		Uuid                string          `json:"uuid"`
		Name                string          `json:"name"`
		Description         *string         `json:"description"`
		OrgUuid             string          `json:"org_uuid"`
		SpatialFilterMode   *string         `json:"spatial_filter_mode"`
		Prefilter           json.RawMessage `json:"prefilter"`
		Geometry            json.RawMessage `json:"geometry"`
		Style               json.RawMessage `json:"style"`
		Header              json.RawMessage `json:"header"`
		Footer              json.RawMessage `json:"footer"`
		WebLogoUuid         *string         `json:"web_logo_uuid"`
		BackgroundImageUuid *string         `json:"background_image_uuid"`
	}

	err := h.DbPool.QueryRow(
		ctx,
		app.UranusInstance.SqlGetPortal,
		portalUuid,
	).Scan(
		&portal.Uuid,
		&portal.Name,
		&portal.Description,
		&portal.OrgUuid,
		&portal.SpatialFilterMode,
		&portal.Prefilter,
		&portal.Geometry,
		&portal.Style,
		&portal.Header,
		&portal.Footer,
		&portal.WebLogoUuid,
		&portal.BackgroundImageUuid,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusBadRequest, "get portal failed")
		return
	}

	apiRequest.Success(http.StatusOK, portal, "")
}
