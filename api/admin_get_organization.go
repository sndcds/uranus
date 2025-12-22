package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) AdminGetOrganization(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()

	organizationId := gc.Param("organizationId")
	if organizationId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "organization Id is required"})
		return
	}

	query := app.Singleton.SqlGetAdminOrganization
	rows, err := pool.Query(ctx, query, organizationId)
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
