package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
	event, selectedDate, furtherDates, err :=
		h.LoadEventByDateIdentifier(gc.Request.Context(), eventUuid, dateUuid, userUuid, lang)

	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	event.Date = selectedDate
	event.FurtherDates = furtherDates

	apiRequest.Success(http.StatusOK, event)
}

func (h *ApiHandler) GetEventByDate(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-event-by-date")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	eventDateRequest, ok := h.ResolveEventDateRequest(gc, apiRequest)
	if !ok {
		apiRequest.InternalServerError()
		return
	}

	eventUuid := eventDateRequest.EventUUID
	dateUuid := eventDateRequest.DateUUID
	lang := eventDateRequest.Lang

	// Load everything via shared function
	event, selectedDate, furtherDates, err := h.LoadEventByDateIdentifier(
		ctx,
		eventUuid,
		dateUuid,
		userUuid,
		lang,
	)

	if err != nil {
		if err.Error() == "event not found" {
			apiRequest.NotFound("event not found")
			return
		}

		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	// Attach dates back to event
	event.Date = selectedDate
	event.FurtherDates = furtherDates

	apiRequest.SetMeta("event_date_count", len(furtherDates)+1)

	apiRequest.Success(http.StatusOK, event)
}

func (h *ApiHandler) LoadEventByDateIdentifier(
	ctx context.Context,
	eventUuid string,
	dateUuid string,
	userUuid string,
	lang string,
) (model.EventDetails, *model.EventDate, []model.EventDate, error) {

	var event model.EventDetails
	var selectedDate *model.EventDate
	var furtherDates []model.EventDate

	// Resolve allowed statuses
	usedStatuses := publicStatuses

	if len(userUuid) > 0 {
		permissions, err := h.GetUserEventOrganizerPermissions(ctx, userUuid, eventUuid)
		if err != nil {
			return event, nil, nil, err
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
		return event, nil, nil, err
	}
	defer eventRow.Close()

	if !eventRow.Next() {
		return event, nil, nil, fmt.Errorf("event not found")
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
	)
	if err != nil {
		return event, nil, nil, err
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
		return event, nil, nil, err
	}
	defer dateRows.Close()

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
			return event, nil, nil, err
		}

		edd.Slug = BuildDateSlug(edd.StartDate, edd.StartTime)

		// Enrich logos
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

		//
		if edd.AccessibilityFlags != nil {
			mask, err := strconv.ParseInt(*edd.AccessibilityFlags, 10, 64)
			if err != nil {
				return event, nil, nil, err
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

	return event, selectedDate, furtherDates, nil
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
