package api_admin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func OrganizerEventsHandler(gc *gin.Context) {
	fmt.Println("Organizer events handler called")
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	type EventWithVenue struct {
		EventID       int     `json:"event_id"`
		EventTitle    string  `json:"event_title"`
		EventSubtitle string  `json:"event_subtitle"`
		OrganizerID   int     `json:"event_organizer_id"`
		StartDate     *string `json:"start_date"`
		StartTime     *string `json:"start_time"`
		EndDate       *string `json:"end_date"`
		EndTime       *string `json:"end_time"`
		VenueID       int     `json:"venue_id"`
		VenueName     string  `json:"venue_name"`
		SpaceID       *int    `json:"space_id,omitempty"`
		SpaceName     *string `json:"space_name,omitempty"`
		VenueLon      float64 `json:"venue_lon"`
		VenueLat      float64 `json:"venue_lat"`
	}

	idStr := gc.Param("id")
	fmt.Println("idStr:", idStr)
	organizerId, err := strconv.Atoi(idStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "invalid organizer id"})
		return
	}
	fmt.Println("organizerId:", organizerId)

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
	fmt.Println("startDate:", startDate)

	rows, err := pool.Query(ctx, app.Singleton.SqlAdminOrganizerEvents, organizerId, startDate)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	fmt.Println("rows:", rows)

	var events []EventWithVenue
	for rows.Next() {
		var e EventWithVenue
		err := rows.Scan(
			&e.EventID,
			&e.EventTitle,
			&e.EventSubtitle,
			&e.OrganizerID,
			&e.StartDate,
			&e.StartTime,
			&e.EndDate,
			&e.EndTime,
			&e.VenueID,
			&e.VenueName,
			&e.SpaceID,
			&e.SpaceName,
			&e.VenueLon,
			&e.VenueLat,
		)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		events = append(events, e)
	}

	if len(events) == 0 {
		gc.JSON(http.StatusNoContent, gin.H{"error": "no events found"})
		return
	}

	gc.JSON(http.StatusOK, events)
}
