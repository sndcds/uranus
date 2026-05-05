package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

var publicStatuses = []string{
	"released",
	"cancelled",
	"deferred",
	"rescheduled",
}

func (h *ApiHandler) GetEventByDateUuid(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-event-by-date-id")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Required("eventUuid is required")
		return
	}
	apiRequest.SetMeta("event_uuid", eventUuid)

	dateUuid := gc.Param("dateUuid")
	if dateUuid == "" {
		apiRequest.Required("dateUuid is required")
		return
	}
	apiRequest.SetMeta("date_uuid", dateUuid)

	usedStatuses := publicStatuses
	hasUserUuid := len(userUuid) > 0
	if hasUserUuid {
		permissions, err := h.GetUserEventOrganizerPermissions(gc, userUuid, eventUuid)
		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}
		if permissions.HasAny(app.UserPermEditEvent | app.UserPermDeleteEvent | app.UserPermReleaseEvent | app.UserPermViewEventInsights) {
			usedStatuses = []string{
				"draft",
				"review",
				"released",
				"cancelled",
				"deferred",
				"rescheduled",
			}
		}
	}

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	// Query event-level data without event dates
	eventRow, err := h.DbPool.Query(ctx, app.UranusInstance.SqlGetEvent, eventUuid, lang, usedStatuses)
	if err != nil {
		debugf(err.Error())
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
	var orgLogosJSON []byte
	var eventTypesJSON []byte
	var eventLinksJSON []byte

	err = eventRow.Scan(
		&event.Uuid,
		&event.ReleaseStatus,
		&event.ContentLanguage,
		&event.Title,
		&event.Subtitle,
		&event.Description,
		&event.Summary,
		&event.ParticipationInfo,
		&event.OnlineLink,
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
		&event.TicketFlags,
		&event.TicketLink,
		&event.VisitorInfoFlags,
		&event.OrgUuid,
		&event.OrgName,
		&event.OrgWebLink,
		&orgLogosJSON,
		&imageJSON,
		&eventTypesJSON,
		&eventLinksJSON,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.DatabaseError()
		return
	}

	// Unmarshal organization logos
	if len(orgLogosJSON) > 0 && string(orgLogosJSON) != "null" {
		err = json.Unmarshal(orgLogosJSON, &event.OrgLogos)
		if err != nil {
			apiRequest.SetMeta("logo_error", err.Error())
		}
	}

	// Unmarshal image JSON
	if len(imageJSON) > 0 {
		var image model.Image
		err = json.Unmarshal(imageJSON, &image)
		if err != nil {
			apiRequest.SetMeta("image_error", "invalid JSON")
		} else {
			event.Image = &image
		}
	}

	// Unmarshal event types
	if len(eventTypesJSON) > 0 {
		var eventTypes []model.EventType
		err = json.Unmarshal(eventTypesJSON, &eventTypes)
		if err == nil {
			event.EventTypes = eventTypes
		}
	}

	// Unmarshal event URLs
	if len(eventLinksJSON) > 0 {
		var eventLinks []model.WebLink
		err = json.Unmarshal(eventLinksJSON, &eventLinks)
		if err == nil {
			event.EventLinks = eventLinks
		}
	}

	// Query all event dates
	dateRows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlGetEventDates, eventUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer dateRows.Close()

	var selectedDate *model.EventDate
	var furtherDates []model.EventDate

	for dateRows.Next() {
		var edd model.EventDate
		err := dateRows.Scan(
			&edd.Uuid,
			&edd.EventUuid,
			&edd.EventReleaseStatus,
			&edd.StartDate,
			&edd.StartTime,
			&edd.EndDate,
			&edd.EndTime,
			&edd.EntryTime,
			&edd.Duration,
			&edd.VenueUuid,
			&edd.VenueName,
			&edd.VenueStreet,
			&edd.VenueHouseNumber,
			&edd.VenuePostalCode,
			&edd.VenueCity,
			&edd.VenueCountry,
			&edd.VenueState,
			&edd.VenueLon,
			&edd.VenueLat,
			&edd.VenueWebLink,
			&edd.VenueLogoImageUuid,
			&edd.VenueLightThemeLogoImageUuid,
			&edd.VenueDarkThemeLogoImageUuid,
			&edd.SpaceUuid,
			&edd.SpaceName,
			&edd.TotalCapacity,
			&edd.SeatingCapacity,
			&edd.BuildingLevel,
			&edd.SpaceWebLink,
			&edd.AccessibilityFlags,
			&edd.AccessibilitySummary,
			&edd.AccessibilityInfo,
		)
		if err != nil {
			apiRequest.InternalServerError()
			return
		}

		// Generate VenueLogoUrl if logo exists
		if edd.VenueLogoImageUuid != nil {
			url := ImageUrl(*edd.VenueLogoImageUuid)
			edd.VenueLogoUrl = &url
		}
		if edd.VenueLightThemeLogoImageUuid != nil {
			url := ImageUrl(*edd.VenueLightThemeLogoImageUuid)
			edd.VenueLightThemeLogoUrl = &url
		}
		if edd.VenueDarkThemeLogoImageUuid != nil {
			url := ImageUrl(*edd.VenueDarkThemeLogoImageUuid)
			edd.VenueDarkThemeLogoUrl = &url
		}

		if edd.Uuid == dateUuid {
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
