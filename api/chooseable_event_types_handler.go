package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func ChoosableEventTypesHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	db := app.Singleton.MainDbPool
	sql := app.Singleton.SqlChoosableEventTypes

	langStr := gc.Param("lang")
	if langStr == "" {
		langStr = "en"
	}

	rows, err := db.Query(ctx, sql, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type EventType struct {
		TypeId int    `json:"type_id"`
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
