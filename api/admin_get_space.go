package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetSpace(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()

	spaceId := gc.Param("spaceId")
	if spaceId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Space ID is required"})
		return
	}

	query := app.Singleton.SqlGetAdminSpace
	rows, err := pool.Query(ctx, query, spaceId)
	if err != nil {
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
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]interface{}, len(values))
	for i, fd := range fieldDescriptions {
		result[string(fd.Name)] = values[i]
	}

	gc.JSON(http.StatusOK, result)
}
