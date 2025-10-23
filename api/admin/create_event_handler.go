package api_admin

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type EventDataIncoming struct {
	OrganizerId int     `json:"organizer_id"`
	VenueId     int     `json:"venue_id"`
	SpaceId     *int    `json:"space_id"`
	Title       *string `json:"title"`
	Subtitle    *string `json:"subtitle"`
	Description string  `json:"description"`
	TeaserText  *string `json:"teaser_text"`

	TypeGenrePairs []struct {
		TypeId  int `json:"type_id"`
		GenreId int `json:"genre_id"`
	} `json:"types"`

	Dates []struct {
		StartDate string  `json:"start_date"`
		EndDate   *string `json:"end_date"`
		StartTime string  `json:"start_time"`
		EndTime   string  `json:"end_time"`
		EntryTime *string `json:"entry_time"`
		SpaceId   *int    `json:"space_id"`
		AllDay    bool    `json:"all_day"`
	} `json:"dates"`
}

func CreateEventHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool
	dbSchema := app.Singleton.Config.DbSchema

	{
		// Read the raw body
		bodyBytes, err := io.ReadAll(gc.Request.Body)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}

		// Print raw JSON to console (or log)
		fmt.Println("Raw JSON:", string(bodyBytes))

		// Reassign body so Gin can still bind it
		gc.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	var incoming EventDataIncoming
	if err := gc.ShouldBindJSON(&incoming); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ok := validate(incoming)
	if !ok {
		gc.JSON(http.StatusUnprocessableEntity, gin.H{"error": "The request is semantically invalid."})
		return
	}

	printDebug(incoming)

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Basic Event Information
	sqlEvent := `
		INSERT INTO {{schema}}.event (
			organizer_id,
			space_id,
			title,
			subtitle,
			description,
			teaser_text
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	sql := strings.Replace(sqlEvent, "{{schema}}", dbSchema, 1)

	var eventId int
	err = tx.QueryRow(ctx, sql,
		incoming.OrganizerId,
		incoming.SpaceId,
		incoming.Title,
		incoming.Subtitle,
		incoming.Description,
		incoming.TeaserText,
	).Scan(&eventId)
	if err != nil {
		_ = tx.Rollback(ctx)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event: %v", err)})
		return
	}

	// Event Dates
	sqlDate := `
		INSERT INTO {{schema}}.event_date (
			event_id,
			space_id,
			start,
			"end",
			entry_time,
			all_day
		) VALUES ($1,$2,$3,$4,$5,$6)`
	sql = strings.Replace(sqlDate, "{{schema}}", dbSchema, 1)

	for _, d := range incoming.Dates {
		// Combine StartDate + StartTime
		start, errStart := time.Parse("2006-01-02 15:04", d.StartDate+" "+d.StartTime)
		if errStart != nil {
			_ = tx.Rollback(ctx)
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid start datetime: %v", errStart)})
			return
		}

		var end *time.Time
		if d.EndDate != nil && d.EndTime != "" {
			t, errEnd := time.Parse("2006-01-02 15:04", *d.EndDate+" "+d.EndTime)
			if errEnd != nil {
				_ = tx.Rollback(ctx)
				gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid end datetime: %v", errEnd)})
				return
			}
			end = &t
		}

		_, err = tx.Exec(ctx, sql, eventId, d.SpaceId, start, end, d.EntryTime, d.AllDay)
		if err != nil {
			_ = tx.Rollback(ctx)
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event_date: %v", err)})
			return
		}
	}

	// Insert Type + Genre pairs
	queryTemplate := `
		INSERT INTO {{schema}}.event_type_links (event_id, type_id, genre_id)
		VALUES ($1, $2, $3)`
	query := strings.Replace(queryTemplate, "{{schema}}", dbSchema, 1)

	for _, pair := range incoming.TypeGenrePairs {
		_, err := tx.Exec(ctx, query, eventId, pair.TypeId, pair.GenreId)
		if err != nil {
			_ = tx.Rollback(ctx)
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("failed to insert type-genre pair: %v", err),
			})
			return
		}
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusCreated, gin.H{"event_id": eventId})
}

func validate(incoming EventDataIncoming) bool {
	return true
}

func printDebug(incoming EventDataIncoming) {
	fmt.Println("OrganizerId:", incoming.OrganizerId)
	fmt.Println("VenueId:", incoming.VenueId)
	if incoming.SpaceId != nil {
		fmt.Println("SpaceId:", *incoming.SpaceId)
	}
	if incoming.Title != nil {
		fmt.Println("Title:", *incoming.Title)
	}
	if incoming.Subtitle != nil {
		fmt.Println("Subtitle:", *incoming.Subtitle)
	}
	fmt.Println("Description:", incoming.Description)
	if incoming.TeaserText != nil {
		fmt.Println("Teaser:", *incoming.TeaserText)
	}

	fmt.Println("Types length:", len(incoming.TypeGenrePairs))
	for i, pair := range incoming.TypeGenrePairs {
		fmt.Printf("Type %d: type_id=%d, genre_id=%d\n", i+1, pair.TypeId, pair.GenreId)
	}

	fmt.Println("Dates length:", len(incoming.Dates))

	for i, d := range incoming.Dates {
		fmt.Printf("Date %d:\n", i+1)
		fmt.Println("  StartDate:", d.StartDate)
		if d.EndDate != nil {
			fmt.Println("  EndDate:", *d.EndDate)
		}
		fmt.Println("  StartTime:", d.StartTime)
		fmt.Println("  EndTime:", d.EndTime)
		if d.EntryTime != nil {
			fmt.Println("  EntryTime:", *d.EntryTime)
		}
		if d.SpaceId != nil {
			fmt.Println("  SpaceId:", *d.SpaceId)
		}
		fmt.Println("  AllDay:", d.AllDay)
	}
}

/*
func CreateEventHandler(gc *gin.Context) {
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
}
*/
