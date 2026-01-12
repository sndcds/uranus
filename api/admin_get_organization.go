package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

	organizationId := gc.Param("organizationId")
	if organizationId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "organization Id is required"})
		return
	}

	query := app.UranusInstance.SqlGetAdminOrganization
	rows, err := h.DbPool.Query(ctx, query, organizationId, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	if !rows.Next() {
		gc.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	fieldDescriptions := rows.FieldDescriptions()
	values, err := rows.Values()
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]interface{}, len(values))
	for i, fd := range fieldDescriptions {
		result[string(fd.Name)] = values[i]
	}

	gc.JSON(http.StatusOK, result)
}
