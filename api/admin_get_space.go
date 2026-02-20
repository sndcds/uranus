package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: User must be authenticated.
// The endpoint returns space details only if the authenticated user
// is linked to the space (via the SQL query).
// PermissionChecks: Enforced in SQL; no additional checks needed in Go.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetSpace(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-get-space")

	spaceId, ok := ParamInt(gc, "spaceId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "spaceId is required")
		return
	}
	apiRequest.SetMeta("space_id", spaceId)

	rows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetSpace, spaceId, userId)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	if !rows.Next() {
		apiRequest.Error(http.StatusNotFound, "space not found")
		return
	}

	fieldDescriptions := rows.FieldDescriptions()
	values, err := rows.Values()
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	result := make(map[string]interface{}, len(values))
	for i, fd := range fieldDescriptions {
		result[string(fd.Name)] = values[i]
	}

	apiRequest.Success(http.StatusOK, result, "space successfully retrieved")
}
