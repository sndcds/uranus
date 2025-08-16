package model

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"log"
	"net/http"
	"time"
)

type Event struct {
	Id    int    `json:"event_id"`
	Title string `json:"title"`
}

func NewEvent() Event {
	return Event{}
}

func (event Event) Print() {
	fmt.Println("Event:")
	fmt.Println("  id:", event.Id)
	fmt.Println("  title:", event.Title)
}

func GetEventsByUserId(app app.Uranus, ctx *gin.Context, userId int) ([]Event, error) {

	query := `
		SELECT
			e.id AS event_id,
			e.title AS event_title,
			v.name AS event_venue_name,
			ed.id AS event_date_id,
			MIN(ed.start) AS event_start_first,
			MAX(ed.start) AS event_start_last,
			CASE 
				WHEN ur.event = TRUE THEN TRUE
				ELSE FALSE
			END AS can_edit
		FROM
			app.event e
		JOIN
			app.venue v ON v.id = e.venue_id
		JOIN
			app.user_venue_links uvl ON uvl.venue_id = v.id
		JOIN
			app.event_date ed ON ed.event_id = e.id
		JOIN
			app.user_role ur ON ur.id = uvl.user_role_id
		WHERE
			uvl.user_id = $1
			AND ed.start >= NOW()
		GROUP BY
			ed.start, e.id, ed.id, v.id, v.name, ur.event
		ORDER BY
			ed.start`

	rows, err := app.MainDbPool.Query(context.Background(), query, userId)
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var events []Event

	for rows.Next() {
		var event = NewEvent()
		var venueName string
		var eventDateId int
		var eventDateStartFirst time.Time
		var eventDateStartLast time.Time
		var canEdit bool

		err := rows.Scan(&event.Id, &event.Title, &venueName, &eventDateId, &eventDateStartFirst, &eventDateStartLast, &canEdit)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read organizer data"})
			return nil, fmt.Errorf("failed to read event data: %w", err)
		}
		events = append(events, event)
	}

	// Check for any errors during row iteration
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating rows"})
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Return the slice of organizers
	return events, nil
}
