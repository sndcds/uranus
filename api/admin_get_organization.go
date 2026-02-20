package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: User must be authenticated.
// The endpoint only returns the organization if the authenticated user
// is linked to it via the user_organization_link table.
// Purpose: Retrieves details of a specific organization for authorized users.
// PermissionChecks: Unnecessary.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetOrganization(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-get-organization")

	organizationId := gc.Param("organizationId")
	if organizationId == "" {
		apiRequest.Error(http.StatusBadRequest, "organizationId is required")
		return
	}

	query := app.UranusInstance.SqlGetAdminOrganization
	rows, err := h.DbPool.Query(ctx, query, organizationId, userId)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "query failed (#1)")
		return
	}
	defer rows.Close()

	if !rows.Next() {
		apiRequest.Error(http.StatusNotFound, "organization not found")
		return
	}

	fieldDescriptions := rows.FieldDescriptions()
	values, err := rows.Values()
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "query failed (#2)")
		return
	}

	data := make(map[string]interface{}, len(values))
	for i, fd := range fieldDescriptions {
		data[string(fd.Name)] = values[i]
	}

	apiRequest.Success(http.StatusOK, data, "organization found")
}
