package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetOrganizerEvents(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()

	type EventType struct {
		TypeID    int     `json:"type_id"`
		TypeName  string  `json:"type_name"`
		GenreID   int     `json:"genre_id"`
		GenreName *string `json:"genre_name"`
	}

	type EventWithVenue struct {
		EventId            int         `json:"event_id"`
		EventTitle         string      `json:"event_title"`
		EventSubtitle      *string     `json:"event_subtitle"`
		EventOrganizerId   int         `json:"event_organizer_id"`
		EventOrganizerName *string     `json:"event_organizer_name"`
		StartDate          *string     `json:"start_date"`
		StartTime          *string     `json:"start_time"`
		EndDate            *string     `json:"end_date"`
		EndTime            *string     `json:"end_time"`
		VenueId            int         `json:"venue_id"`
		VenueName          string      `json:"venue_name"`
		SpaceId            *int        `json:"space_id,omitempty"`
		SpaceName          *string     `json:"space_name,omitempty"`
		VenueLon           float64     `json:"venue_lon"`
		VenueLat           float64     `json:"venue_lat"`
		ImageId            *int        `json:"image_id"`
		EventTypes         []EventType `json:"event_types"`
	}

	organizerIdStr := gc.Param("organizerId")
	organizerId, err := strconv.Atoi(organizerIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organizer id"})
		return
	}

	startStr := gc.Query("start")
	var startDate time.Time
	if startStr != "" {
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			startDate = time.Now()
		}
	} else {
		startDate = time.Now()
	}

	rows, err := pool.Query(ctx, app.Singleton.SqlAdminOrganizerEvents, organizerId, startDate, "de")
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var eventTypesData []byte
	var events []EventWithVenue
	for rows.Next() {
		var e EventWithVenue
		err := rows.Scan(
			&e.EventId,
			&e.EventTitle,
			&e.EventSubtitle,
			&e.EventOrganizerId,
			&e.EventOrganizerName,
			&e.StartDate,
			&e.StartTime,
			&e.EndDate,
			&e.EndTime,
			&e.VenueId,
			&e.VenueName,
			&e.SpaceId,
			&e.SpaceName,
			&e.VenueLon,
			&e.VenueLat,
			&e.ImageId,
			&eventTypesData,
		)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if len(eventTypesData) > 0 {
			_ = json.Unmarshal(eventTypesData, &e.EventTypes)
		}
		events = append(events, e)
	}

	if len(events) == 0 {
		gc.JSON(http.StatusNoContent, gin.H{"error": "no events found"})
		return
	}

	gc.JSON(http.StatusOK, events)
}
