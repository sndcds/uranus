package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) AdminGetEvent(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	langStr := gc.DefaultQuery("lang", "en")

	// Query event info (without dates)
	eventRows, err := h.DbPool.Query(ctx, app.Singleton.SqlGetAdminEvent, eventId, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer eventRows.Close()

	var event map[string]interface{}
	if eventRows.Next() {
		fieldDescs := eventRows.FieldDescriptions()
		columns := make([]string, len(fieldDescs))
		for i, fd := range fieldDescs {
			columns[i] = string(fd.Name)
		}

		values, err := eventRows.Values()
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		event = make(map[string]interface{}, len(values))
		for i, col := range columns {
			event[col] = values[i]
		}

	} else {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	if err := eventRows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Query all event_dates for this event
	dateRows, err := h.DbPool.Query(ctx, app.Singleton.SqlGetAdminEventDates, eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer dateRows.Close()

	var eventDates []map[string]interface{}
	dateFieldDescs := dateRows.FieldDescriptions()
	dateColumns := make([]string, len(dateFieldDescs))
	for i, fd := range dateFieldDescs {
		dateColumns[i] = string(fd.Name)
	}

	for dateRows.Next() {
		values, err := dateRows.Values()
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rowMap := make(map[string]interface{}, len(values))
		for i, col := range dateColumns {
			rowMap[col] = values[i]
		}

		eventDates = append(eventDates, rowMap)
	}

	if err := dateRows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Merge event_dates into event map
	event["event_dates"] = eventDates

	gc.JSON(http.StatusOK, event)
}
