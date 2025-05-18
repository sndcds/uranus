package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"net/http"
	"strings"
	"time"
)

type EventDateData struct {
	Start              *time.Time `json:"start"`
	End                *time.Time `json:"end"`
	AccessibilityFlags []int      `json:"accessibility_flags"`
	VisitorInfoFlags   []int      `json:"visitor_info_flags"`
	SpaceId            *int       `json:"space_id"`
	EntryTime          *string    `json:"entry_time"`
}

type EventData struct {
	OrganizerId int             `json:"organizer_id"`
	SpaceId     *int            `json:"space_id"`
	Title       *string         `json:"title"`
	Subtitle    *string         `json:"subtitle"`
	Description *string         `json:"description"`
	Teaser      *string         `json:"teaser"`
	ImageURL    *string         `json:"image_url,omitempty"`
	EventTypes  []int           `json:"event_types"`
	GenreTypes  []int           `json:"genre_types"`
	EventDates  []EventDateData `json:"event_dates"`
}

func CreateEventHandler(gc *gin.Context) {
	var eventData EventData
	if err := gc.ShouldBindJSON(&eventData); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jsonBytes, err := json.MarshalIndent(eventData, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
	} else {
		fmt.Println(string(jsonBytes))
	}

	db := app.Singleton.MainDb
	dbSchema := app.Singleton.Config.DbSchema

	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Could not start transaction"})
		return
	}
	defer tx.Rollback(ctx)

	// Insert basic event information

	sqlTemplate := `
		INSERT INTO {{schema}}.event (
			organizer_id,
			space_id,
			title,
			subtitle,
			description,
			teaser_text
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) RETURNING id`

	sqlQuery := strings.Replace(sqlTemplate, "{{schema}}", dbSchema, 1)
	fmt.Println("sqlQuery:", sqlQuery)

	var eventId int
	err = tx.QueryRow(ctx, sqlQuery,
		eventData.OrganizerId,
		eventData.SpaceId,
		eventData.Title,
		eventData.Subtitle,
		eventData.Description,
		eventData.Teaser).Scan(&eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert event"})
		return
	}

	// Insert event dates

	for _, eventDate := range eventData.EventDates {
		accessibilityFlags := app.CombineFlags(eventDate.AccessibilityFlags)
		visitorFlags := app.CombineFlags(eventDate.VisitorInfoFlags)

		columns := []string{"event_id", "start"}
		args := []interface{}{eventId, eventDate.Start}

		if eventDate.End != nil {
			columns = append(columns, "end")
			args = append(args, eventDate.End)
		}
		columns = append(columns, "accessibility_flags", "visitor_info_flags")
		args = append(args, accessibilityFlags, visitorFlags)

		if eventDate.SpaceId != nil {
			columns = append(columns, "space_id")
			args = append(args, *eventDate.SpaceId)
		}
		if eventDate.EntryTime != nil {
			columns = append(columns, "entry_time")
			args = append(args, *eventDate.EntryTime)
		}

		// Construct placeholder string
		placeholders := make([]string, len(args))
		for i := range args {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}

		sqlTemplate := `
			INSERT INTO {{schema}}.event_date ({{columns}}) VALUES ({{values}}) RETURNING id`
		sqlQuery := strings.Replace(sqlTemplate, "{{schema}}", dbSchema, 1)
		sqlQuery = strings.Replace(sqlQuery, "{{columns}}", strings.Join(columns, ", "), 1)
		sqlQuery = strings.Replace(sqlQuery, "{{values}}", strings.Join(placeholders, ", "), 1)
		fmt.Println("sqlQuery:", sqlQuery)
		fmt.Println("placeholders:", placeholders)
		fmt.Println("columns:", columns)
		fmt.Println("args:", args)

		_, err := tx.Exec(ctx, sqlQuery, args...)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert event date"})
			return
		}
	}

	/*
		// TODO: !!!
		if eventData.ImageURL != nil {
			var imageId int
			err = tx.QueryRow(ctx,
				`INSERT INTO image (url) VALUES ($1) RETURNING id`,
				*eventData.ImageURL,
			).Scan(&imageId)
			if err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert image"})
				return
			}

			_, err = tx.Exec(ctx,
				`INSERT INTO event_image_links (event_id, image_id) VALUES ($1, $2)`,
				eventId, imageId,
			)
			if err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link image"})
				return
			}
		}
	*/

	// Event types
	{
		queryTemplate := `INSERT INTO {{schema}}.event_type_links (event_id, type_id) VALUES ($1, $2)`
		query := strings.Replace(queryTemplate, "{{schema}}", dbSchema, 1)
		fmt.Println("query:", query)
		for _, typeId := range eventData.EventTypes {
			fmt.Println("eventId", eventId, "typeId:", typeId)
			_, err := tx.Exec(ctx, query, eventId, typeId)
			if err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert event types"})
				return
			}
		}
	}

	// Genre types
	{
		queryTemplate := `INSERT INTO {{schema}}.event_genre_links (event_id, type_id) VALUES ($1, $2)`
		query := strings.Replace(queryTemplate, "{{schema}}", dbSchema, 1)
		fmt.Println("query:", query)
		for _, genreId := range eventData.GenreTypes {
			fmt.Println("eventId", eventId, "genreId:", genreId)
			_, err := tx.Exec(ctx, query, eventId, genreId)
			if err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert genres"})
				return
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	gc.JSON(http.StatusCreated, gin.H{"event_id": eventId})
}
