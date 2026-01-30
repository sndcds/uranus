package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetEvent(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiResponseType := "admin_event"

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		JSONError(gc, apiResponseType, http.StatusBadRequest, "eventId is required")
		return
	}

	lang := gc.DefaultQuery("lang", "en")

	permission := app.PermEditEvent | app.PermViewEventInsights
	row := h.DbPool.QueryRow(ctx, app.UranusInstance.SqlAdminGetEvent, eventId, lang, userId, permission)

	// Basic Event
	var event model.AdminEvent
	err := row.Scan(
		&event.Id,
		&event.ExternalId,
		&event.SourceUrl,
		&event.ReleaseStatus,
		&event.ReleaseDate,
		&event.ContentLanguage,
		&event.OrganizationId,
		&event.OrganizationName,
		&event.Title,
		&event.Subtitle,
		&event.Description,
		&event.Summary,
		&event.Tags,
		&event.OccasionType,
		&event.VenueId,
		&event.VenueName,
		&event.VenueStreet,
		&event.VenueHouseNumber,
		&event.VenuePostalCode,
		&event.VenueCity,
		&event.VenueCountry,
		&event.VenueState,
		&event.VenueLon,
		&event.VenueLat,
		&event.SpaceId,
		&event.SpaceName,
		&event.SpaceTotalCapacity,
		&event.SpaceSeatingCapacity,
		&event.SpaceBuildingLevel,
		&event.LocationId,
		&event.LocationName,
		&event.LocationStreet,
		&event.LocationHouseNumber,
		&event.LocationPostalCode,
		&event.LocationCity,
		&event.LocationCountry,
		&event.LocationState,
		&event.OnlineEventUrl,
		&event.MeetingPoint,
		&event.Languages,
		&event.ParticipationInfo,
		&event.MinAge,
		&event.MaxAge,
		&event.MaxAttendees,
		&event.PriceType,
		&event.MinPrice,
		&event.MaxPrice,
		&event.TicketAdvance,
		&event.TicketRequired,
		&event.RegistrationRequired,
		&event.Currency,
		&event.CurrencyName,
		&event.Custom,
		&event.Style,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			JSONError(gc, apiResponseType, http.StatusNotFound, "event not found")
			return
		}
		JSONDatabaseError(gc, apiResponseType)
		return
	}

	// Event Types
	rows, _ := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventTypes, eventId, lang)
	defer rows.Close()
	for rows.Next() {
		var et model.EventType
		rows.Scan(&et.Type, &et.TypeName, &et.Genre, &et.GenreName)
		event.EventTypes = append(event.EventTypes, et)
	}

	// Event Images
	rows, _ = h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventImages, eventId)
	defer rows.Close()
	for rows.Next() {
		var img model.Image
		rows.Scan(&img.Id, &img.Identifier, &img.FocusX, &img.FocusY, &img.Alt, &img.Copyright, &img.Creator, &img.License)
		img.Url = fmt.Sprintf("%s/api/image/%d", h.Config.BaseApiUrl, img.Id)
		event.Images = append(event.Images, img)
	}

	// Event Links
	rows, _ = h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventLinks, eventId)
	defer rows.Close()
	for rows.Next() {
		var link model.WebLink
		rows.Scan(&link.Id, &link.Type, &link.Url, &link.Title)
		event.EventLinks = append(event.EventLinks, link)
	}

	// Dates
	rows, _ = h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventDates, eventId)
	defer rows.Close()

	for rows.Next() {
		var date model.AdminEventDate
		err := rows.Scan(
			&date.Id,
			&date.EventId,
			&date.StartDate,
			&date.StartTime,
			&date.EndDate,
			&date.EndTime,
			&date.EntryTime,
			&date.Duration,
			&date.AccessibilityInfo,
			&date.VisitorInfoFlags,
			&date.DateVenueId,
			&date.VenueId,
			&date.VenueName,
			&date.VenueStreet,
			&date.VenueHouseNumber,
			&date.VenuePostalCode,
			&date.VenueCity,
			&date.VenueCountry,
			&date.VenueState,
			&date.VenueLon,
			&date.VenueLat,
			&date.VenueUrl,
			&date.SpaceId,
			&date.SpaceName,
			&date.SpaceTotalCapacity,
			&date.SpaceSeatingCapacity,
			&date.SpaceBuildingLevel,
			&date.SpaceUrl,
		)

		if err != nil {
			JSONDatabaseError(gc, apiResponseType)
			return
		}

		event.EventDates = append(event.EventDates, date)
	}

	JSONSuccess(gc, apiResponseType, event, nil)
}
