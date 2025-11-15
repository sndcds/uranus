package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type EventIncomingLocation struct {
	Street      string  `json:"street" binding:"required"`
	PostalCode  string  `json:"postal_code" binding:"required"`
	City        string  `json:"city" binding:"required"`
	CountryCode string  `json:"country_code" binding:"required"`
	Name        *string `json:"name"`
	HouseNumber *string `json:"house_number"`
	StateCode   *string `json:"state_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Description *string `json:"description"`
}

type EventIncomingTypeGenrePair struct {
	TypeId  int  `json:"type_id" binding:"required"`
	GenreId *int `json:"genre_id"`
}

type EventIncomingDate struct {
	StartDate string                `json:"start_date" binding:"required"`
	StartTime string                `json:"start_time" binding:"required"`
	EndDate   *string               `json:"end_date"`
	EndTime   *string               `json:"end_time"`
	EntryTime *string               `json:"entry_time"`
	VenueId   *int                  `json:"venue_id"`
	SpaceId   *int                  `json:"space_id"`
	AllDay    *bool                 `json:"all_day"`
	Location  EventIncomingLocation `json:"location"`
}

type EventDataIncoming struct {
	OrganizerId          *int    `json:"organizer_id" binding:"required"`
	Title                string  `json:"title" binding:"required"`
	Description          string  `json:"description" binding:"required"`
	VenueId              *int    `json:"venue_id"`
	SpaceId              *int    `json:"space_id"`
	ExternalId           *int    `json:"external_id"`
	Subtitle             *string `json:"subtitle"`
	TeaserText           *string `json:"teaser_text"`
	ParticipationInfo    *string `json:"participation_info"`
	MeetingPoint         *string `json:"meeting_point"`
	SourceUrl            *string `json:"source_url"`
	MinAge               *int    `json:"min_age"`
	MaxAge               *int    `json:"max_age"`
	MaxAttendees         *int    `json:"max_attendees"`
	Custom               *string `json:"custom"`
	Style                *string `json:"style"`
	ReleaseStatusId      *int    `json:"release_status_id"`
	ReleaseDate          *string `json:"release_date"`
	PriceTypeId          *int    `json:"price_type_id"`
	TicketAdvance        *bool   `json:"ticket_advance"`
	TicketRequired       *bool   `json:"ticket_required"`
	RegistrationRequired *bool   `json:"registration_required"`
	CurrencyCode         *string `json:"currency_code"`
	OnlineEventUrl       *string `json:"online_event_url"`
	OcationTypeId        *int    `json:"ocation_type_id"`

	Location       *EventIncomingLocation       `json:"location"`
	TypeGenrePairs []EventIncomingTypeGenrePair `json:"types"`
	Dates          []EventIncomingDate          `json:"dates" binding:"required"`

	Languages []string `json:"languages"`
	Tags      []string `json:"tags"`
}

func (h *ApiHandler) AdminCreateEvent(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	userId := gc.GetInt("user-id")

	// Read the raw body
	bodyBytes, err := io.ReadAll(gc.Request.Body)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Reassign body so Gin can still bind it
	gc.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Unmarschal JSON
	body, err := io.ReadAll(gc.Request.Body)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read request body"})
		return
	}

	if len(body) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	var incomingEvent EventDataIncoming
	if err := json.Unmarshal(body, &incomingEvent); err != nil {
		var ute *json.UnmarshalTypeError
		var se *json.SyntaxError
		var iue *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &se):
			gc.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid JSON syntax (at offset %d)", se.Offset),
			})
			return

		case errors.As(err, &ute):
			field := ute.Field
			if field == "" {
				field = "(unknown field)"
			}
			gc.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid type for field %q: expected %v but got %v", field, ute.Type, ute.Value),
				"hint":  "check numeric and boolean values — don't quote numbers or booleans",
			})
			return

		case errors.Is(err, io.EOF):
			gc.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
			return

		case errors.As(err, &iue):
			// This is a programming bug, not client error
			log.Printf("Internal unmarshal error: %v", err)
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return

		default:
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	validationErr := incomingEvent.Validate()
	if validationErr != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
		return
	}

	incomingEvent.printDebug()

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locationId *int
	if incomingEvent.HasLocation() {
		// Insert the event location first

		locationSql := `
			INSERT INTO {{schema}}.event_location (
				name,
				street,
				house_number,
				postal_code,
				city,
				country_code,
				state_code,
			    wkb_geometry,
				description,
				create_user_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, ST_SetSRID(ST_MakePoint($8, $9), 4326), $10, $11)
			RETURNING id`
		locationSql = strings.Replace(locationSql, "{{schema}}", h.Config.DbSchema, 1)
		err = tx.QueryRow(ctx, locationSql,
			incomingEvent.Location.Name,
			incomingEvent.Location.Street,
			incomingEvent.Location.HouseNumber,
			incomingEvent.Location.PostalCode,
			incomingEvent.Location.City,
			incomingEvent.Location.CountryCode,
			incomingEvent.Location.StateCode,
			incomingEvent.Location.Longitude,
			incomingEvent.Location.Latitude,
			incomingEvent.Location.Description,
			userId,
		).Scan(&locationId)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event location: %v", err)})
			return
		}
	}

	// Basic Event Information
	sqlEvent := `
		INSERT INTO {{schema}}.event (
			organizer_id,
			venue_id,
			space_id,
			location_id,
			title,
			subtitle,
			description,
			teaser_text,
		  	languages
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`
	sql := strings.Replace(sqlEvent, "{{schema}}", h.Config.DbSchema, 1)

	var eventId int
	err = tx.QueryRow(ctx, sql,
		incomingEvent.OrganizerId,
		incomingEvent.VenueId,
		incomingEvent.SpaceId,
		locationId,
		incomingEvent.Title,
		incomingEvent.Subtitle,
		incomingEvent.Description,
		incomingEvent.TeaserText,
		incomingEvent.Languages,
	).Scan(&eventId)
	if err != nil {
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
		) VALUES ($1, $2, $3, $4, $5, $6)`
	sql = strings.Replace(sqlDate, "{{schema}}", h.Config.DbSchema, 1)

	for _, d := range incomingEvent.Dates {
		// Combine StartDate + StartTime
		start, errStart := time.Parse("2006-01-02 15:04", d.StartDate+" "+d.StartTime)
		if errStart != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid start datetime: %v", errStart)})
			return
		}

		var end *time.Time
		if d.EndDate != nil && *d.EndTime != "" {
			t, errEnd := time.Parse("2006-01-02 15:04", *d.EndDate+" "+*d.EndTime)
			if errEnd != nil {
				gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid end datetime: %v", errEnd)})
				return
			}
			end = &t
		}

		_, err = tx.Exec(ctx, sql, eventId, d.SpaceId, start, end, d.EntryTime, d.AllDay)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event_date: %v", err)})
			return
		}
	}

	// Insert Type + Genre pairs
	queryTemplate := `
		INSERT INTO {{schema}}.event_type_link (event_id, type_id, genre_id)
		VALUES ($1, $2, $3)`
	query := strings.Replace(queryTemplate, "{{schema}}", h.Config.DbSchema, 1)

	for _, pair := range incomingEvent.TypeGenrePairs {
		_, err := tx.Exec(ctx, query, eventId, pair.TypeId, pair.GenreId)
		if err != nil {
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

func (e *EventDataIncoming) printDebug() {

	fmt.Println("VenueId:", e.VenueId)
	if e.SpaceId != nil {
		fmt.Println("SpaceId:", *e.SpaceId)
	}
	fmt.Println("Title:", e.Title)
	if e.Subtitle != nil {
		fmt.Println("Subtitle:", *e.Subtitle)
	}
	fmt.Println("Description:", e.Description)
	if e.TeaserText != nil {
		fmt.Println("Teaser:", *e.TeaserText)
	}

	fmt.Println("Types length:", len(e.TypeGenrePairs))
	for i, pair := range e.TypeGenrePairs {
		fmt.Printf("Type %d: type_id=%d, genre_id=%d\n", i+1, pair.TypeId, pair.GenreId)
	}

	fmt.Println("Dates length:", len(e.Dates))

	for i, d := range e.Dates {
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

	if e.HasLocation() {
		fmt.Println("Location:", e.Location)
	}
}

// Validate validates the EventDataIncoming struct
func (e *EventDataIncoming) Validate() error {
	var errs []string

	if e.OrganizerId == nil {
		errs = append(errs, "organizer_id is required")
	} else if *e.OrganizerId < 0 {
		errs = append(errs, "organizer_id must be >= 0")
	}

	if strings.TrimSpace(e.Title) == "" {
		errs = append(errs, "title is required")
	}
	if strings.TrimSpace(e.Description) == "" {
		errs = append(errs, "description is required")
	}

	// Dates
	if len(e.Dates) == 0 {
		errs = append(errs, "at least one date is required")
	} else {
		for i, date := range e.Dates {
			if strings.TrimSpace(date.StartDate) == "" {
				errs = append(errs, fmt.Sprintf("dates[%d].start_date is required", i))
			}
			if strings.TrimSpace(date.StartTime) == "" {
				errs = append(errs, fmt.Sprintf("dates[%d].start_time is required", i))
			}
		}
	}

	if !e.HasVenue() && !e.HasLocation() {
		errs = append(errs, "event must have either venue_id or location")
	} else if e.HasVenue() && e.HasLocation() {
		errs = append(errs, "event cannot have both venue_id and location")
	}

	// Age constraints
	if e.MinAge != nil {
		if *e.MinAge < 0 {
			errs = append(errs, "min_age cannot be less than 0")
		}
		if e.MaxAge != nil && *e.MinAge > *e.MaxAge {
			errs = append(errs, "min_age cannot be greater than max_age")
		}
	} else if e.MaxAge != nil {
		if *e.MaxAge < 0 {
			errs = append(errs, "max_age cannot be less than 0")
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (e *EventDataIncoming) HasVenue() bool {
	return e.VenueId != nil
}

func (e *EventDataIncoming) HasLocation() bool {
	return e.Location != nil
}
