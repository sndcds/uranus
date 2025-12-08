package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetOrganizerEvents(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")
	langStr := gc.DefaultQuery("lang", "en")

	type EventType struct {
		TypeID    int     `json:"type_id"`
		TypeName  string  `json:"type_name"`
		GenreID   int     `json:"genre_id"`
		GenreName *string `json:"genre_name"`
	}

	type OrganizerEvent struct {
		EventId            int         `json:"event_id"`
		EventDateId        int         `json:"event_date_id"`
		EventTitle         string      `json:"event_title"`
		EventSubtitle      *string     `json:"event_subtitle"`
		EventOrganizerId   int         `json:"event_organizer_id"`
		EventOrganizerName *string     `json:"event_organizer_name"`
		StartDate          *string     `json:"start_date"`
		StartTime          *string     `json:"start_time"`
		EndDate            *string     `json:"end_date"`
		EndTime            *string     `json:"end_time"`
		ReleaseStatusId    *int        `json:"release_status_id"`
		ReleaseStatusName  *string     `json:"release_status_name"`
		ReleaseDate        *string     `json:"release_date"`
		VenueId            *int        `json:"venue_id"`
		VenueName          *string     `json:"venue_name"`
		SpaceId            *int        `json:"space_id,omitempty"`
		SpaceName          *string     `json:"space_name,omitempty"`
		VenueLon           *float64    `json:"venue_lon"`
		VenueLat           *float64    `json:"venue_lat"`
		ImageId            *int        `json:"image_id"`
		EventTypes         []EventType `json:"event_types"`
		CanEditEvent       bool        `json:"can_edit_event"`
		CanDeleteEvent     bool        `json:"can_delete_event"`
		CanReleaseEvent    bool        `json:"can_release_event"`
		TimeSeriesIndex    int         `json:"time_series_index"`
		TimeSeries         int         `json:"time_series"`
	}

	organizerId, ok := ParamInt(gc, "organizerId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organizer id"})
		return
	}

	fmt.Println("organizerId: ", organizerId)

	startStr := gc.Query("start")
	var err error
	var startDate time.Time
	if startStr != "" {
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			startDate = time.Now()
		}
	} else {
		startDate = time.Now()
	}

	rows, err := pool.Query(ctx, app.Singleton.SqlAdminGetOrganizerEvents, organizerId, startDate, langStr, userId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var eventTypesData []byte
	var events []OrganizerEvent
	for rows.Next() {
		fmt.Printf("....")
		var e OrganizerEvent
		err := rows.Scan(
			&e.EventId,
			&e.EventDateId,
			&e.EventTitle,
			&e.EventSubtitle,
			&e.EventOrganizerId,
			&e.EventOrganizerName,
			&e.StartDate,
			&e.StartTime,
			&e.EndDate,
			&e.EndTime,
			&e.ReleaseStatusId,
			&e.ReleaseStatusName,
			&e.ReleaseDate,
			&e.VenueId,
			&e.VenueName,
			&e.SpaceId,
			&e.SpaceName,
			&e.VenueLon,
			&e.VenueLat,
			&e.ImageId,
			&eventTypesData,
			&e.CanEditEvent,
			&e.CanDeleteEvent,
			&e.CanReleaseEvent,
			&e.TimeSeriesIndex,
			&e.TimeSeries,
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
