package api_admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func GetAdminEventHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	langStr := gc.DefaultQuery("lang", "en")

	query := app.Singleton.SqlGetAdminEvent

	rows, err := pool.Query(ctx, query, eventId, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columnNames[i] = string(fd.Name)
	}

	var result map[string]interface{}

	if rows.Next() {
		values, err := rows.Values()
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		result = make(map[string]interface{}, len(values))
		for i, col := range columnNames {
			result[col] = values[i]
		}
	} else {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, result)
}
