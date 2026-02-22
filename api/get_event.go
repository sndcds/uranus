package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) GetEventByDateId(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "get-event-by-date-id")

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "eventId is required")
		return
	}
	apiRequest.SetMeta("event_id", eventId)

	dateId, ok := ParamInt(gc, "dateId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "dateId is required")
		return
	}
	apiRequest.SetMeta("date_id", dateId)

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	// Query event-level data without event dates
	eventRow, err := h.DbPool.Query(ctx, app.UranusInstance.SqlGetEvent, eventId, lang)
	if err != nil {
		debugf("GetEventByDateId err 1: %v", err)
		apiRequest.DatabaseError()
		return
	}
	defer eventRow.Close()

	if !eventRow.Next() {
		apiRequest.NotFound("event not found")
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
		&event.VisitorInfoFlags,
		&event.OrganizationId,
		&event.OrganizationName,
		&event.OrganizationUrl,
		&imageJSON,
		&eventTypesJSON,
		&eventLinksJSON,
	)
	if err != nil {
		debugf("GetEventByDateId err 2: %v", err)
		apiRequest.DatabaseError()
		return
	}

	// Unmarshal image JSON
	if len(imageJSON) > 0 {
		var img model.Image
		err := json.Unmarshal(imageJSON, &img)
		if err != nil {
			apiRequest.SetMeta("image_error", "invalid JSON")
		} else {
			event.Image = &img
		}
	}

	// Unmarshal event types
	if len(eventTypesJSON) > 0 {
		var types []model.EventType
		err = json.Unmarshal(eventTypesJSON, &types)
		if err == nil {
			event.EventTypes = types
		}
	}

	// Unmarshal event URLs
	if len(eventLinksJSON) > 0 {
		var links []model.WebLink
		err = json.Unmarshal(eventLinksJSON, &links)
		if err == nil {
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
			&edd.VenueName,
			&edd.VenueStreet,
			&edd.VenueHouseNumber,
			&edd.VenuePostalCode,
			&edd.VenueCity,
			&edd.VenueCountry,
			&edd.VenueState,
			&edd.VenueLon,
			&edd.VenueLat,
			&edd.VenueWebsiteUrl,
			&edd.VenueLogoImageId,
			&edd.VenueLightThemeLogoImageId,
			&edd.VenueDarkThemeLogoImageId,
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
			apiRequest.InternalServerError()
			return
		}

		// Generate VenueLogoUrl if logo exists
		if edd.VenueLogoImageId != nil {
			url := fmt.Sprintf("%s/api/image/%d", app.UranusInstance.Config.BaseApiUrl, *edd.VenueLogoImageId)
			edd.VenueLogoUrl = &url
		}
		if edd.VenueLightThemeLogoImageId != nil {
			url := fmt.Sprintf("%s/api/image/%d", app.UranusInstance.Config.BaseApiUrl, *edd.VenueLightThemeLogoImageId)
			edd.VenueLightThemeLogoUrl = &url
		}
		if edd.VenueDarkThemeLogoImageId != nil {
			url := fmt.Sprintf("%s/api/image/%d", app.UranusInstance.Config.BaseApiUrl, *edd.VenueDarkThemeLogoImageId)
			edd.VenueDarkThemeLogoUrl = &url
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
	apiRequest.SetMeta("event_date_count", len(furtherDates)+1)

	apiRequest.Success(http.StatusOK, event, "")
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
