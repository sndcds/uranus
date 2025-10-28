package api_admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func GetAdminOrganizerHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	organizerId := gc.Param("organizerId")
	if organizerId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "organizer ID is required"})
		return
	}

	query := app.Singleton.SqlGetAdminOrganizer
	rows, err := pool.Query(ctx, query, organizerId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	if !rows.Next() {
		gc.JSON(http.StatusNotFound, gin.H{"error": "organizer not found"})
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
