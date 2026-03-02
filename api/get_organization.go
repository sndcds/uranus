package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetOrganization(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "get-organization")

	organizationId, ok := ParamInt(gc, "organizationId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "organizationId is required")
		return
	}
	apiRequest.SetMeta("organizationId", organizationId)

	query := app.UranusInstance.SqlGetOrganization
	rows, err := h.DbPool.Query(ctx, query, organizationId)
	if err != nil {
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
