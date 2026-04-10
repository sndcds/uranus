package api

import (
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
		apiRequest.Error(http.StatusBadRequest, "eventUuid is required")
		return
	}

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	permission := app.PermEditEvent | app.PermViewEventInsights
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
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			apiRequest.Error(http.StatusNotFound, "event not found")
			return
		}
		debugf(err.Error())
		apiRequest.InternalServerError()
		apiRequest.SetMeta("error_type", "event")
		return
	}

	// Event Types
	rows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventTypes, eventUuid, lang)
	if err != nil {
		debugf(err.Error())
		apiRequest.SetMeta("error_type", "event-types")
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()
	for rows.Next() {
		var et model.EventType
		rows.Scan(&et.Type, &et.TypeName, &et.Genre, &et.GenreName)
		event.EventTypes = append(event.EventTypes, et)
	}

	// Event Images
	rows, err = h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventImages, eventUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		apiRequest.SetMeta("error_type", "event-images")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var img model.Image
		rows.Scan(&img.Uuid, &img.Identifier, &img.FocusX, &img.FocusY, &img.Alt, &img.Copyright, &img.Creator, &img.License)
		img.Url = ImageUrl(img.Uuid)
		event.Images = append(event.Images, img)
	}

	// Event Links
	rows, err = h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventLinks, eventUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		apiRequest.SetMeta("error_type", "event-links")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var link model.WebLink
		rows.Scan(&link.Label, &link.Type, &link.Url)
		event.EventLinks = append(event.EventLinks, link)
	}

	// Dates
	rows, err = h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventDates, eventUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		apiRequest.SetMeta("error_type", "event-dates")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var date model.AdminEventDate
		err := rows.Scan(
			&date.Uuid,
			&date.EventUuid,
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
			debugf(err.Error())
			apiRequest.DatabaseError()
			return
		}

		event.EventDates = append(event.EventDates, date)
	}

	apiRequest.Success(http.StatusOK, event, "")
}
