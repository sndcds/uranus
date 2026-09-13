package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetEvent(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-event")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Required("Parameter eventUuid is required")
		return
	}

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	event, err := h.loadAdminEvent(ctx, eventUuid, lang, userUuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apiRequest.Error(http.StatusNotFound, "Event not found")
			return
		}
		debugf("%v", err)
		apiRequest.InternalServerError()
		return
	}
	apiRequest.Success(http.StatusOK, event)
}

// loadAdminEvent applies the same organizer permissions to details and quality reports.
// Any failed relation query aborts the load to avoid evaluating incomplete data.
func (h *ApiHandler) loadAdminEvent(ctx context.Context, eventUuid, lang, userUuid string) (model.AdminEvent, error) {
	permission := app.UserPermEditEvent | app.UserPermViewEventInsights

	row := h.DbPool.QueryRow(ctx, app.UranusInstance.SqlAdminGetEvent, eventUuid, lang, userUuid, permission)

	// Basic Event
	var event model.AdminEvent
	err := row.Scan(
		&event.Uuid,
		&event.ExternalId,
		&event.SourceLink,
		&event.ReleaseStatus,
		&event.ReleaseDate,
		&event.Categories,
		&event.ContentLanguage,
		&event.OrgUuid,
		&event.OrgName,
		&event.Title,
		&event.Subtitle,
		&event.Description,
		&event.Summary,
		&event.Tags,
		&event.OccasionType,
		&event.VenueUuid,
		&event.VenueName,
		&event.VenueStreet,
		&event.VenueHouseNumber,
		&event.VenuePostalCode,
		&event.VenueCity,
		&event.VenueCountry,
		&event.VenueState,
		&event.VenueLon,
		&event.VenueLat,
		&event.SpaceUuid,
		&event.SpaceName,
		&event.SpaceTotalCapacity,
		&event.SpaceSeatingCapacity,
		&event.SpaceBuildingLevel,
		&event.OnlineLink,
		&event.RegistrationLink,
		&event.RegistrationEmail,
		&event.RegistrationPhone,
		&event.RegistrationDeadline,
		&event.MeetingPoint,
		&event.Languages,
		&event.ParticipationInfo,
		&event.MinAge,
		&event.MaxAge,
		&event.MaxAttendees,
		&event.PriceType,
		&event.MinPrice,
		&event.MaxPrice,
		&event.TicketFlags,
		&event.TicketLink,
		&event.Currency,
		&event.CurrencyName,
		&event.VisitorInfoFlags,
		&event.Custom,
		&event.Style,
		&event.LogoMode,
		&event.CanRelease,
	)

	if err != nil {
		return event, fmt.Errorf("event: %w", err)
	}

	// Event Types
	rows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventTypes, eventUuid, lang)
	if err != nil {
		return event, fmt.Errorf("event relations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var et model.EventType
		if err := rows.Scan(&et.Type, &et.TypeName, &et.Genre, &et.GenreName); err != nil {
			return event, fmt.Errorf("event types: %w", err)
		}
		event.EventTypes = append(event.EventTypes, et)
	}

	if err := rows.Err(); err != nil {
		return event, fmt.Errorf("event relations: %w", err)
	}

	// Event Images
	rows, err = h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventImages, eventUuid)
	if err != nil {
		return event, fmt.Errorf("event relations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var img model.Image
		if err := rows.Scan(&img.Uuid, &img.Identifier, &img.FocusX, &img.FocusY, &img.Alt, &img.Copyright, &img.Creator, &img.License, &img.Width, &img.Height); err != nil {
			return event, fmt.Errorf("event images: %w", err)
		}
		img.Url = ImageUrl(img.Uuid)
		event.Images = append(event.Images, img)
	}

	if err := rows.Err(); err != nil {
		return event, fmt.Errorf("event relations: %w", err)
	}

	// Event Links
	rows, err = h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventLinks, eventUuid)
	if err != nil {
		return event, fmt.Errorf("event relations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var link model.WebLink
		if err := rows.Scan(&link.Label, &link.Type, &link.Url); err != nil {
			return event, fmt.Errorf("event links: %w", err)
		}
		event.EventLinks = append(event.EventLinks, link)
	}

	if err := rows.Err(); err != nil {
		return event, fmt.Errorf("event relations: %w", err)
	}

	// Dates
	rows, err = h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventDates, eventUuid)
	if err != nil {
		return event, fmt.Errorf("event relations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var date model.AdminEventDate
		err := rows.Scan(
			&date.Uuid,
			&date.EventUuid,
			&date.ReleaseStatus,
			&date.StartDate,
			&date.StartTime,
			&date.EndDate,
			&date.EndTime,
			&date.EntryTime,
			&date.Duration,
			&date.AllDay,
			&date.AccessibilityInfo,
			&date.VenueUuid,
			&date.VenueName,
			&date.VenueStreet,
			&date.VenueHouseNumber,
			&date.VenuePostalCode,
			&date.VenueCity,
			&date.VenueCountry,
			&date.VenueState,
			&date.VenueLon,
			&date.VenueLat,
			&date.VenueLink,
			&date.SpaceUuid,
			&date.SpaceName,
			&date.SpaceTotalCapacity,
			&date.SpaceSeatingCapacity,
			&date.SpaceBuildingLevel,
			&date.SpaceLink,
		)

		if err != nil {
			return event, fmt.Errorf("event dates: %w", err)
		}

		event.EventDates = append(event.EventDates, date)
	}

	if err := rows.Err(); err != nil {
		return event, fmt.Errorf("event relations: %w", err)
	}

	return event, nil
}
