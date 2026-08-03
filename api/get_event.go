package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

func (h *ApiHandler) GetEvent(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-event")

	eventUuid := gc.Param("eventUuid")
	userUuid := h.userUuid(gc)
	lang := gc.DefaultQuery("lang", "en")

	dateUuid := ""
	event, err := h.LoadEventByDateIdentifier(
		gc.Request.Context(),
		eventUuid,
		dateUuid,
		userUuid,
		lang)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, event)
}

func (h *ApiHandler) GetEventByDate(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-event-by-date")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	eventDateRequest, ok := h.ResolveEventDateRequest(gc, apiRequest)
	if !ok {
		return
	}

	eventUuid := eventDateRequest.EventUuid
	dateUuid := eventDateRequest.DateUuid
	lang := eventDateRequest.Lang

	// Load everything via shared function

	event, err := h.LoadEventByDateIdentifier(
		ctx,
		eventUuid,
		dateUuid,
		userUuid,
		lang)
	if err != nil {
		if err.Error() == "event not found" {
			apiRequest.NotFound("event not found")
			return
		}

		apiRequest.InternalServerError()
		return
	}

	apiRequest.SetMeta("event_date_count", len(event.FurtherDates)+1)
	apiRequest.Success(http.StatusOK, event)
}

func (h *ApiHandler) LoadEventByDateIdentifier(
	ctx context.Context,
	eventUuid string,
	dateUuid string,
	userUuid string,
	lang string,
) (model.EventDetails, error) {

	var event model.EventDetails
	var selectedDate *model.EventDate
	var furtherDates []model.EventDate

	// Resolve allowed statuses

	usedStatuses := publicStatuses

	if len(userUuid) > 0 {
		permissions, err := h.GetUserEventOrganizerPermissions(ctx, userUuid, eventUuid)
		if err != nil {
			return event, err
		}

		if permissions.HasAny(
			app.UserPermEditEvent |
				app.UserPermDeleteEvent |
				app.UserPermReleaseEvent |
				app.UserPermViewEventInsights,
		) {
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

	// Load event (main query)

	eventRow, err := h.DbPool.Query(ctx,
		app.UranusInstance.SqlGetEvent,
		eventUuid,
		lang,
		usedStatuses,
	)
	if err != nil {
		return event, err
	}
	defer eventRow.Close()

	if !eventRow.Next() {
		return event, fmt.Errorf("event not found")
	}

	var (
		imagesJSON     []byte
		orgLogosJSON   []byte
		eventTypesJSON []byte
		eventLinksJSON []byte
	)

	err = eventRow.Scan(
		&event.Uuid,
		&event.ReleaseStatus,
		&event.ContentLanguage,
		&event.Title,
		&event.Subtitle,
		&event.Description,
		&event.Summary,
		&event.SourceUrl,
		&event.ParticipationInfo,
		&event.OnlineLink,
		&event.RegistrationLink,
		&event.RegistrationEmail,
		&event.RegistrationPhone,
		&event.RegistrationDeadline,
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
		&imagesJSON,
		&eventTypesJSON,
		&eventLinksJSON,
		&event.LogoMode,
	)
	if err != nil {
		return event, err
	}

	// Cleanup ticket flags

	var allowedTicketFlags = map[string]struct{}{
		"advance_ticket":          {},
		"presale_fee_applies":     {},
		"on_site_ticket_sales":    {},
		"reduced_price_available": {},
	}

	event.TicketFlags = app.FilterStrings(event.TicketFlags, allowedTicketFlags)

	// Unmarshal JSON fields

	if len(orgLogosJSON) > 0 && string(orgLogosJSON) != "null" {
		_ = json.Unmarshal(orgLogosJSON, &event.OrgLogos)
	}

	if len(imagesJSON) > 0 && string(imagesJSON) != "null" {
		var images map[string]model.Image
		if err := json.Unmarshal(imagesJSON, &images); err == nil {
			event.Images = images
		}
	}

	if len(eventTypesJSON) > 0 {
		var eventTypes []model.EventType
		if err := json.Unmarshal(eventTypesJSON, &eventTypes); err == nil {
			event.EventTypes = eventTypes
		}
	}

	if len(eventLinksJSON) > 0 {
		var eventLinks []model.WebLink
		if err := json.Unmarshal(eventLinksJSON, &eventLinks); err == nil {
			event.EventLinks = eventLinks
		}
	}

	// Load event dates

	dateRows, err := h.DbPool.Query(ctx,
		app.UranusInstance.SqlGetEventDates,
		eventUuid,
	)
	if err != nil {
		return event, err
	}
	defer dateRows.Close()

	for dateRows.Next() {
		var edd model.EventDate
		var venueLogos []byte

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
			&venueLogos,
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
			return event, err
		}

		edd.Slug = BuildDateSlug(edd.StartDate, edd.StartTime)

		// Unmarshal JSON fields

		if len(venueLogos) > 0 && string(venueLogos) != "null" {
			_ = json.Unmarshal(venueLogos, &edd.VenueLogos)
		}

		//
		if edd.AccessibilityFlags != nil {
			mask, err := strconv.ParseInt(*edd.AccessibilityFlags, 10, 64)
			if err != nil {
				return event, err
			}
			edd.AccessibilityLabels = h.Accessibility.LabelsForMask(mask, lang)
		}

		// Split selected vs others

		if edd.Uuid == dateUuid {
			tmp := edd
			selectedDate = &tmp
		} else {
			furtherDates = append(furtherDates, edd)
		}
	}

	// When no date is requested, select the nearest upcoming date and remove it from further dates.

	if selectedDate == nil && len(furtherDates) > 0 {
		today := time.Now().In(time.Local).Format("2006-01-02")

		selectedIndex := -1
		for i := range furtherDates {
			if furtherDates[i].StartDate >= today {
				selectedIndex = i
				break
			}
		}

		// No upcoming date: use the latest past date.
		if selectedIndex == -1 {
			selectedIndex = len(furtherDates) - 1
		}

		chosenDate := furtherDates[selectedIndex]
		selectedDate = &chosenDate

		furtherDates = append(
			furtherDates[:selectedIndex],
			furtherDates[selectedIndex+1:]...,
		)
	} else {
		event.DateMatch = true
	}

	event.Date = selectedDate
	event.FurtherDates = furtherDates

	return event, nil
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
