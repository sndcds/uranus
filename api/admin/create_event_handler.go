package api_admin

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type EventDataIncoming struct {
	OrganizerId  int     `json:"organizer_id"`
	VenueId      int     `json:"venue_id"`
	SpaceId      *int    `json:"space_id"`
	Title        *string `json:"title"`
	Subtitle     *string `json:"subtitle"`
	Description  string  `json:"description"`
	TeaserText   *string `json:"teaser_text"`
	EventTypeIds []int   `json:"event_type_ids"`
	GenreTypeIds []int   `json:"genre_type_ids"`
	Dates        []struct {
		StartDate string  `json:"start_date"`
		EndDate   *string `json:"end_date"`
		StartTime string  `json:"start_time"`
		EndTime   string  `json:"end_time"`
		EntryTime *string `json:"entry_time"`
		SpaceId   *int    `json:"space_id"`
		AllDay    bool    `json:"all_day"`
	} `json:"dates"`
}

func validate(incoming EventDataIncoming) bool {

	// Debug information
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
	fmt.Println("EventTypeIds length:", len(incoming.EventTypeIds))
	fmt.Println("GenreTypeIds length:", len(incoming.GenreTypeIds))
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

	return true
}

func CreateEventHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool
	dbSchema := app.Singleton.Config.DbSchema

	var incoming EventDataIncoming
	if err := gc.ShouldBindJSON(&incoming); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate incoming data
	ok := validate(incoming)
	if !ok {
		gc.JSON(http.StatusUnprocessableEntity, gin.H{"error": "The request is semantically invalid."})
		return
	}

	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}

	// Ensure rollback if any error occurs
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Insert basic event information
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

	// Insert event dates
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

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	fmt.Println("Inserted event ID:", eventId)
	gc.JSON(http.StatusCreated, gin.H{"event_id": eventId})
}

func CreateEventHandler2(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool
	dbSchema := app.Singleton.Config.DbSchema

	var incoming EventDataIncoming
	if err := gc.ShouldBindJSON(&incoming); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ok := validate(incoming)
	if !ok {
		gc.JSON(http.StatusUnprocessableEntity, gin.H{"error": "The request is syntactically correct (valid JSON) but semantically invalid."})
		return
	}

	// Begin transaction
	tx, err := pool.Begin(gc)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(gc)
		}
	}()

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

	var eventId int
	err = tx.QueryRow(ctx, sqlQuery,
		incoming.OrganizerId,
		incoming.SpaceId,
		incoming.Title,
		incoming.Subtitle,
		incoming.Description,
		incoming.TeaserText).Scan(&eventId)
	if err != nil {
		_ = tx.Rollback(gc)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event: %v", err)})
		return
	}
	/*
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
			// fmt.Println("sqlQuery:", sqlQuery)
			// fmt.Println("placeholders:", placeholders)
			// fmt.Println("columns:", columns)
			// fmt.Println("args:", args)

			_, err := tx.Exec(ctx, sqlQuery, args...)
			if err != nil {
				fmt.Println("Failed to insert event 2")
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

			// Event types
			{
				queryTemplate := `INSERT INTO {{schema}}.event_type_links (event_id, type_id) VALUES ($1, $2)`
				query := strings.Replace(queryTemplate, "{{schema}}", dbSchema, 1)
				// fmt.Println("query:", query)
				for _, typeId := range eventData.EventTypes {
					// fmt.Println("eventId", eventId, "typeId:", typeId)
					_, err := tx.Exec(ctx, query, eventId, typeId)
					if err != nil {
						fmt.Println("Failed to insert event 3")
						gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert event types"})
						return
					}
				}
			}

			// Genre types
			{
				queryTemplate := `INSERT INTO {{schema}}.event_genre_links (event_id, type_id) VALUES ($1, $2)`
				query := strings.Replace(queryTemplate, "{{schema}}", dbSchema, 1)
				// fmt.Println("query:", query)
				for _, genreId := range eventData.GenreTypes {
					// fmt.Println("eventId", eventId, "genreId:", genreId)
					_, err := tx.Exec(ctx, query, eventId, genreId)
					if err != nil {
						fmt.Println("Failed to insert event 4")
						gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert genres"})
						return
					}
				}
			}

			if err := tx.Commit(ctx); err != nil {
				fmt.Println("Failed to insert event 5")
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
				return
			}
	*/

	if err = tx.Commit(gc); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	fmt.Println("eventId:", eventId)

	gc.JSON(http.StatusCreated, gin.H{"event_id": eventId})
}
