package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) GetEventByDateId(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
		return
	}

	dateId, ok := ParamInt(gc, "dateId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "dateId is required"})
		return
	}

	lang := gc.DefaultQuery("lang", "en")

	// Query event-level data without event dates
	eventRow, err := h.DbPool.Query(ctx, app.UranusInstance.SqlGetEvent, eventId, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer eventRow.Close()

	if !eventRow.Next() {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	var event model.EventDetails
	var imageJSON []byte
	var eventTypesJSON []byte
	var eventLinksJSON []byte

	err = eventRow.Scan(
		&event.Id,
		&event.ReleaseStatus,
		&event.Title,
		&event.Subtitle,
		&event.Description,
		&event.Summary,
		&event.ParticipationInfo,
		&event.MeetingPoint,
		&event.Languages,
		&event.Tags,
		&event.MaxAttendees,
		&event.MinAge,
		&event.MaxAge,
		&event.Currency,
		&event.PriceType,
		&event.MinPrice,
		&event.MaxPrice,
		&event.OrganizationId,
		&event.OrganizationName,
		&event.OrganizationUrl,
		&imageJSON,
		&eventTypesJSON,
		&eventLinksJSON,
	)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Unmarshal image JSON
	if len(imageJSON) > 0 {
		var img model.Image
		if err := json.Unmarshal(imageJSON, &img); err == nil {
			event.Image = &img
		}
	}

	// Unmarshal event types
	if len(eventTypesJSON) > 0 {
		var types []model.EventType
		if err := json.Unmarshal(eventTypesJSON, &types); err == nil {
			event.EventTypes = types
		}
	}

	// Unmarshal event URLs
	if len(eventLinksJSON) > 0 {
		var links []model.WebLink
		if err := json.Unmarshal(eventLinksJSON, &links); err == nil {
			event.EventLinks = links
		}
	}

	// Query all event dates
	dateRows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlGetEventDates, eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer dateRows.Close()

	var selectedDate *model.EventDate
	var furtherDates []model.EventDate

	for dateRows.Next() {
		var edd model.EventDate
		err := dateRows.Scan(
			&edd.Id,
			&edd.EventId,
			&edd.EventReleaseStatus,
			&edd.StartDate,
			&edd.StartTime,
			&edd.EndDate,
			&edd.EndTime,
			&edd.EntryTime,
			&edd.Duration,
			&edd.VenueId,
			&edd.LocationName,
			&edd.Street,
			&edd.HouseNumber,
			&edd.PostalCode,
			&edd.City,
			&edd.Country,
			&edd.State,
			&edd.Lon,
			&edd.Lat,
			&edd.VenueWebsiteUrl,
			&edd.VenueLogoImageId,
			&edd.SpaceId,
			&edd.SpaceName,
			&edd.TotalCapacity,
			&edd.SeatingCapacity,
			&edd.BuildingLevel,
			&edd.SpaceWebsiteUrl,
			&edd.AccessibilityFlags,
			&edd.AccessibilitySummary,
			&edd.AccessibilityInfo,
		)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Generate VenueLogoUrl if logo exists
		if edd.VenueLogoImageId != nil {
			url := fmt.Sprintf("%s/api/image/%d", app.UranusInstance.Config.BaseApiUrl, *edd.VenueLogoImageId)
			edd.VenueLogoUrl = &url
		}

		if edd.Id == dateId {
			tmp := edd
			selectedDate = &tmp
		} else {
			furtherDates = append(furtherDates, edd)
		}
	}

	event.Date = selectedDate
	event.FurtherDates = furtherDates

	gc.JSON(http.StatusOK, event)
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
