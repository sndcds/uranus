package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetEvent(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
		return
	}

	lang := gc.DefaultQuery("lang", "en")

	row := h.DbPool.QueryRow(ctx, app.UranusInstance.SqlAdminGetEvent, eventId, lang, userId)

	var event model.AdminEvent
	err := row.Scan(
		&event.EventId,
		&event.Title,
		&event.Subtitle,
		&event.Description,
		&event.Summary,
		&event.ParticipationInfo,
		&event.Languages,
		&event.Tags,
		&event.MeetingPoint,
		&event.ReleaseStatusId,
		&event.ReleaseDate,
		&event.MinAge,
		&event.MaxAge,
		&event.MaxAttendees,
		&event.PriceTypeId,
		&event.MinPrice,
		&event.MaxPrice,
		&event.TicketAdvance,
		&event.TicketRequired,
		&event.RegistrationRequired,
		&event.CurrencyCode,
		&event.CurrencyName,
		&event.OccasionTypeId,
		&event.OnlineEventUrl,
		&event.SourceUrl,
		&event.Image1Id,
		&event.Image2Id,
		&event.Image3Id,
		&event.Image4Id,
		&event.ImageSoMe16To9Id,
		&event.ImageSoMe4To5Id,
		&event.ImageSoMe9To16Id,
		&event.ImageSoMe1To1Id,
		&event.Custom,
		&event.Style,
		&event.OrganizationId,
		&event.OrganizationName,
		&event.VenueId,
		&event.VenueName,
		&event.VenueStreet,
		&event.VenueHouseNumber,
		&event.VenuePostalCode,
		&event.VenueCity,
		&event.VenueCountryCode,
		&event.VenueStateCode,
		&event.VenueLon,
		&event.VenueLat,
		&event.SpaceId,
		&event.SpaceName,
		&event.SpaceTotalCapacity,
		&event.SpaceSeatingCapacity,
		&event.SpaceBuildingLevel,
		&event.SpaceUrl,
		&event.LocationName,
		&event.LocationStreet,
		&event.LocationHouseNumber,
		&event.LocationPostalCode,
		&event.LocationCity,
		&event.LocationCountryCode,
		&event.LocationStateCode,
		&event.EventTypes,
		&event.EventUrls,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// --- Fetch event dates ---
	rows, err := h.DbPool.Query(ctx, app.UranusInstance.SqlAdminGetEventDates, eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var d model.AdminEventDate
		err := rows.Scan(
			&d.EventDateId,
			&d.EventId,
			&d.StartDate,
			&d.StartTime,
			&d.EndDate,
			&d.EndTime,
			&d.EntryTime,
			&d.Duration,
			&d.AccessibilityInfo,
			&d.VisitorInfoFlags,
			&d.DateVenueId,
			&d.VenueId,
			&d.VenueName,
			&d.VenueStreet,
			&d.VenueHouseNumber,
			&d.VenuePostalCode,
			&d.VenueCity,
			&d.VenueCountryCode,
			&d.VenueStateCode,
			&d.VenueLon,
			&d.VenueLat,
			&d.VenueUrl,
			&d.SpaceId,
			&d.SpaceName,
			&d.SpaceTotalCapacity,
			&d.SpaceSeatingCapacity,
			&d.SpaceBuildingLevel,
			&d.SpaceUrl,
		)

		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		event.EventDates = append(event.EventDates, d)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, event)
}
