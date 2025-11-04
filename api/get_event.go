package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetEvent(gc *gin.Context) {
	// Todo: Adopt SQL Query from GetAdminEventHandler
	pool := h.DbPool
	ctx := gc.Request.Context()

	eventId := gc.Param("id")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	langStr := gc.DefaultQuery("lang", "en")
	dateStr := gc.DefaultQuery("date", "")
	fmt.Println("dateStr", dateStr)

	query := app.Singleton.SqlGetEvent

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

		// Add extra property image_path
		imageID := result["image_id"]
		if imageID == nil {
			result["image_path"] = nil // or "" if you prefer
		} else {
			result["image_path"] = fmt.Sprintf(
				"%s/api/image/%v",
				app.Singleton.Config.BaseApiUrl,
				imageID,
			)
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
