package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
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

	portalIdentifier := gc.Param("portalIdentifier")
	if portalIdentifier == "" {
		apiRequest.Required("portalIdentifier is required")
		return
	}

	var condition string

	if grains_uuid.IsValidUuidv7(portalIdentifier) {
		condition = "WHERE uuid = $1::uuid"
	} else {
		condition = "WHERE slug = $1::text"
	}

	query := strings.Replace(app.UranusInstance.SqlGetPortal2, "{{condition}}", condition, 1)

	var portal struct {
		Uuid               string           `json:"uuid"`
		Slug               *string          `json:"slug"`
		OrgUuid            string           `json:"org_uuid"`
		Name               string           `json:"name"`
		Description        *string          `json:"description,omitempty"`
		GeometryMode       *string          `json:"geometry_mode,omitempty"`
		Geometry           json.RawMessage  `json:"geometry,omitempty"`
		Filter             json.RawMessage  `json:"filter,omitempty"`
		FilterType         string           `json:"filter_type"`
		WebLogoUrl         *string          `json:"web_logo_url,omitempty"`
		MainImageUrl       *string          `json:"main_image_url,omitempty"`
		BackgroundImageUrl *string          `json:"background_image_url,omitempty"`
		FooterLogoUrl      *string          `json:"footer_logo_url,omitempty"`
		Config             *json.RawMessage `json:"config,omitempty"`
	}

	err := h.DbPool.QueryRow(
		ctx,
		query,
		portalIdentifier,
	).Scan(
		&portal.Uuid,
		&portal.Slug,
		&portal.OrgUuid,
		&portal.Name,
		&portal.Description,
		&portal.GeometryMode,
		&portal.Geometry,
		&portal.Filter,
		&portal.FilterType,
		&portal.WebLogoUrl,
		&portal.MainImageUrl,
		&portal.BackgroundImageUrl,
		&portal.FooterLogoUrl,
		&portal.Config,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.Error(http.StatusBadRequest, "get portal2 failed")
		return
	}

	apiRequest.Success(http.StatusOK, portal)
}
