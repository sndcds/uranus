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
		apiRequest.Required("uuid is required")
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
		FooterLogoUuid      *string         `json:"footer_logo_uuid"`
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
		&portal.FooterLogoUuid,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusBadRequest, "get portal failed")
		return
	}

	apiRequest.Success(http.StatusOK, portal)
}

func (h *ApiHandler) GetPortal2(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-portal2")
	ctx := gc.Request.Context()

	portalUuid := gc.Param("uuid")
	if portalUuid == "" {
		apiRequest.Required("uuid is required")
		return
	}

	var portal2 struct {
		Uuid               string          `json:"uuid"`
		OrgUuid            string          `json:"org_uuid"`
		Name               string          `json:"name"`
		Description        *string         `json:"description,omitempty"`
		GeometryMode       *string         `json:"geometry_mode,omitempty"`
		Geometry           json.RawMessage `json:"geometry,omitempty"`
		Filter             json.RawMessage `json:"filter,omitempty"`
		FilterType         string          `json:"filter_type"`
		WebLogoUrl         *string         `json:"web_logo_url,omitempty"`
		BackgroundImageUrl *string         `json:"background_image_url,omitempty"`
		FooterLogoUrl      *string         `json:"footer_logo_url,omitempty"`
	}

	err := h.DbPool.QueryRow(
		ctx,
		app.UranusInstance.SqlGetPortal2,
		portalUuid,
	).Scan(
		&portal2.Uuid,
		&portal2.OrgUuid,
		&portal2.Name,
		&portal2.Description,
		&portal2.GeometryMode,
		&portal2.Geometry,
		&portal2.Filter,
		&portal2.FilterType,
		&portal2.WebLogoUrl,
		&portal2.BackgroundImageUrl,
		&portal2.FooterLogoUrl,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusBadRequest, "get portal2 failed")
		return
	}

	apiRequest.Success(http.StatusOK, portal2)
}
