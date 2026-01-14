package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableEventTypes(gc *gin.Context) {
	ctx := gc.Request.Context()
	query := app.UranusInstance.SqlChoosableEventTypes

	lang := gc.DefaultQuery("lang", "en")
	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type EventType struct {
		TypeId int    `json:"id"`
		Name   string `json:"name"`
	}

	var eventTypes []EventType

	for rows.Next() {
		var eventType EventType
		if err := rows.Scan(&eventType.TypeId, &eventType.Name); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		eventTypes = append(eventTypes, eventType)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return JSON directly
	gc.JSON(http.StatusOK, eventTypes)
}
