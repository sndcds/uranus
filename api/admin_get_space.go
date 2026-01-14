package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// PermissionNote: User must be authenticated.
// The endpoint returns space details only if the authenticated user
// is linked to the space (via the SQL query).
// PermissionChecks: Already enforced in SQL; no additional checks needed in Go.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetSpace(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	spaceId := gc.Param("spaceId")
	if spaceId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Space Id is required"})
		return
	}

	query := app.UranusInstance.SqlAdminGetSpace

	rows, err := h.DbPool.Query(ctx, query, spaceId, userId)
	if err != nil {
		fmt.Println("...1")
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	if !rows.Next() {
		gc.JSON(http.StatusNotFound, gin.H{"error": "Space not found"})
		return
	}

	fieldDescriptions := rows.FieldDescriptions()
	values, err := rows.Values()
	if err != nil {
		fmt.Println("...2")
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]interface{}, len(values))
	for i, fd := range fieldDescriptions {
		result[string(fd.Name)] = values[i]
	}

	gc.JSON(http.StatusOK, result)
}
