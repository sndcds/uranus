package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminUpdatePortalFilter(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-portal-filter")
	ctx := gc.Request.Context()

	portalUuid := gc.Param("portalUuid")
	if portalUuid == "" {
		apiRequest.Required("portalUuid is required")
		return
	}

	// Parse JSON body
	var style map[string]any
	if err := gc.ShouldBindJSON(&style); err != nil {
		apiRequest.InvalidJSONInput()
		return
	}

	// Convert to JSON bytes
	styleJSON, err := json.Marshal(style)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to marshal style JSON: "+err.Error())
		return
	}

	query := fmt.Sprintf(`UPDATE %s.portal SET filter = $1::jsonb WHERE uuid = $2::uuid`, h.DbSchema)
	_, err = h.DbPool.Exec(ctx, query, styleJSON, portalUuid)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}
