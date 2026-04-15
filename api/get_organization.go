package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetOrganization(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-organization")
	ctx := gc.Request.Context()

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "parameter orgUuid is required")
		return
	}
	apiRequest.SetMeta("org_uuid", orgUuid)

	query := app.UranusInstance.SqlGetOrganization
	rows, err := h.DbPool.Query(ctx, query, orgUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	if !rows.Next() {
		apiRequest.NotFound("organization not found")
		return
	}

	fieldDescriptions := rows.FieldDescriptions()
	values, err := rows.Values()
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	response := make(map[string]interface{}, len(values))
	for i, fd := range fieldDescriptions {
		if values[i] != nil {
			response[string(fd.Name)] = values[i]
		}
	}

	apiRequest.Success(http.StatusOK, response, "")
}
