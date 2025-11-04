package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetEvent(gc *gin.Context) {
	// TODO: Implement
	// Must return an event with all its dates
	gc.JSON(http.StatusBadRequest, gin.H{"error": "not implemented"})
}

func (h *ApiHandler) GetEventByDateId(gc *gin.Context) {
	// Todo: Adopt SQL Query from GetAdminEventHandler
	pool := h.DbPool
	ctx := gc.Request.Context()

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	dateId := gc.Param("dateId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "date ID is required"})
		return
	}

	fmt.Println("eventId:", eventId)
	fmt.Println("dateId:", dateId)

	langStr := gc.DefaultQuery("lang", "en")

	query := app.Singleton.SqlGetEvent

	rows, err := pool.Query(ctx, query, dateId, langStr)
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
