package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetEvent(gc *gin.Context) {
	// TODO: Implement
	// Must return an event with all its dates
	gc.JSON(http.StatusBadRequest, gin.H{"error": "not implemented"})
}

func (h *ApiHandler) GetEventByDateId(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	dateId, ok := ParamInt(gc, "dateId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "date Id is required"})
		return
	}

	langStr := gc.DefaultQuery("lang", "en")

	// Query event-level data without event dates
	eventRow, err := h.DbPool.Query(ctx, app.Singleton.SqlGetEvent, eventId, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer eventRow.Close()

	if !eventRow.Next() {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	eventFieldDesc := eventRow.FieldDescriptions()
	eventCols := make([]string, len(eventFieldDesc))
	for i, fd := range eventFieldDesc {
		eventCols[i] = string(fd.Name)
	}

	eventData := make(map[string]interface{})
	eventValues, err := eventRow.Values()
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for i, col := range eventCols {
		eventData[col] = eventValues[i]
	}

	// Add image_path if image_id exists
	if imageID, ok := eventData["image_id"]; ok && imageID != nil {
		eventData["image_path"] = fmt.Sprintf("%s/api/image/%v", app.Singleton.Config.BaseApiUrl, imageID)
	} else {
		eventData["image_path"] = nil
	}

	// Query all event dates
	datesQuery := app.Singleton.SqlGetEventDates
	dateRows, err := h.DbPool.Query(ctx, datesQuery, eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer dateRows.Close()

	var eventDates []map[string]interface{}
	dateFieldDesc := dateRows.FieldDescriptions()
	dateCols := make([]string, len(dateFieldDesc))
	for i, fd := range dateFieldDesc {
		dateCols[i] = string(fd.Name)
	}

	for dateRows.Next() {
		fmt.Println(dateRows.Values())
		values, err := dateRows.Values()
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		dateData := make(map[string]interface{}, len(values))
		for i, col := range dateCols {
			dateData[col] = values[i]
		}

		eventDates = append(eventDates, dateData)
	}

	eventDates = app.FilterNilSlice(eventDates)

	// Split event dates into selected date + further dates
	var selectedDate map[string]interface{}
	var furtherDates []map[string]interface{}

	for _, d := range eventDates {
		if intFromAny(d["event_date_id"]) == dateId {
			selectedDate = d
		} else {
			furtherDates = append(furtherDates, d)
		}
	}

	// Add to output
	eventData["date"] = selectedDate
	eventData["further_dates"] = furtherDates
	eventData = app.FilterNilMap(eventData)

	gc.JSON(http.StatusOK, eventData)
}

func intFromAny(v interface{}) int {
	switch t := v.(type) {
	case int32:
		return int(t)
	case int64:
		return int(t)
	case int:
		return t
	}
	return 0
}
