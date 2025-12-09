package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type EventIncomingLocation struct {
	Name        *string `json:"name"`
	Description *string `json:"description" binding:"required"`
	Street      string  `json:"street" binding:"required"`
	HouseNumber *string `json:"house_number"`
	PostalCode  string  `json:"postal_code" binding:"required"`
	City        string  `json:"city" binding:"required"`
	CountryCode string  `json:"country_code" binding:"required"`
	StateCode   *string `json:"state_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type EventIncomingTypeGenrePair struct {
	TypeId  int  `json:"type_id" binding:"required"`
	GenreId *int `json:"genre_id"`
}

type EventIncomingDate struct {
	StartDate string  `json:"start_date" binding:"required"`
	StartTime string  `json:"start_time" binding:"required"`
	EndDate   *string `json:"end_date"`
	EndTime   *string `json:"end_time"`
	EntryTime *string `json:"entry_time"`
	VenueId   *int    `json:"venue_id"`
	SpaceId   *int    `json:"space_id"`
	AllDay    *bool   `json:"all_day"`
}

type EventDataIncoming struct {
	Title                string   `json:"title" binding:"required"`
	Description          string   `json:"description" binding:"required"`
	Subtitle             *string  `json:"subtitle"`
	TeaserText           *string  `json:"teaser_text"`
	Tags                 []string `json:"tags"`
	SourceUrl            *string  `json:"source_url"`
	OnlineEventUrl       *string  `json:"online_event_url"`
	OrganizerId          *int     `json:"organizer_id" binding:"required"`
	VenueId              *int     `json:"venue_id"`
	SpaceId              *int     `json:"space_id"`
	ExternalId           *string  `json:"external_id"`
	ParticipationInfo    *string  `json:"participation_info"`
	OccasionTypeId       *int     `json:"occasion_type_id"`
	Languages            []string `json:"languages"`
	MinAge               *int     `json:"min_age"`
	MaxAge               *int     `json:"max_age"`
	MeetingPoint         *string  `json:"meeting_point"`
	MaxAttendees         *int     `json:"max_attendees"`
	PriceTypeId          *int     `json:"price_type_id"`
	CurrencyCode         *string  `json:"currency_code"`
	TicketAdvance        *bool    `json:"ticket_advance"`
	TicketRequired       *bool    `json:"ticket_required"`
	RegistrationRequired *bool    `json:"registration_required"`
	Custom               *string  `json:"custom"`
	Style                *string  `json:"style"`
	ReleaseStatusId      *int     `json:"release_status_id"`
	ReleaseDate          *string  `json:"release_date"`

	Location       *EventIncomingLocation       `json:"location"`
	Dates          []EventIncomingDate          `json:"dates" binding:"required"`
	TypeGenrePairs []EventIncomingTypeGenrePair `json:"types"`
}

func (h *ApiHandler) AdminCreateEvent(gc *gin.Context) {
	ctx := gc.Request.Context()
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

	// Validation
	validationErr := incomingEvent.Validate()
	if validationErr != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
		return
	}

	if incomingEvent.OrganizerId == nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "organizer ID is required"})
		return
	}

	// Begin transaction
	tx, err := h.DbPool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check if user can create an event with 'incomingEvent.OrganizerId' as the organizer
	organizerPermissions, err := h.GetUserOrganizerPermissions(gc, tx, userId, *incomingEvent.OrganizerId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if organizerPermissions.HasAll(app.PermChooseAsEventOrganizer | app.PermAddEvent) {
		gc.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	// Check if user can create an event in 'incomingEvent.VenueId'
	if incomingEvent.VenueId != nil {
		fmt.Println("GetUserVenuePermissions 1")
		venuePermissions, err := h.GetUserVenuePermissions(
			gc, tx, userId, *incomingEvent.OrganizerId, *incomingEvent.VenueId)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fmt.Println("GetUserVenuePermissions 2")
		if venuePermissions.Has(app.PermChooseVenue) {
			gc.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		fmt.Println("GetUserVenuePermissions 3")
	}

	var locationId *int
	if incomingEvent.hasLocation() {
		// If JSON has a location field, insert the event location first
		locationQuery := `
			INSERT INTO {{schema}}.event_location (
				"name",
				street,
				house_number,
				postal_code,
				city,
				country_code,
				state_code,
			    wkb_geometry,
				description,
			    created_by
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, ST_SetSRID(ST_MakePoint($8, $9), 4326), $10, $11)
			RETURNING id`
		locationQuery = strings.Replace(locationQuery, "{{schema}}", h.Config.DbSchema, 1)
		err = tx.QueryRow(
			ctx, locationQuery,
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

	fmt.Println("Insert event 1")
	// Insert event Information
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
		  	languages,
			created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`
	sql := strings.Replace(sqlEvent, "{{schema}}", h.Config.DbSchema, 1)
	fmt.Println("Insert event 2")

	var eventId int
	err = tx.QueryRow(
		ctx, sql,
		incomingEvent.OrganizerId,
		incomingEvent.VenueId,
		incomingEvent.SpaceId,
		locationId,
		incomingEvent.Title,
		incomingEvent.Subtitle,
		incomingEvent.Description,
		incomingEvent.TeaserText,
		incomingEvent.Languages,
		userId,
	).Scan(&eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event: %v, userId: %d", err, userId)})
		return
	}

	fmt.Println("Insert dates 1")
	// Event Dates
	insertDateQuery := `
		INSERT INTO {{schema}}.event_date (
			event_id,
			venue_id,
			space_id,
			start_date,
			start_time,
			end_date,
			end_time,
			entry_time,
			all_day,
			created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	sql = strings.Replace(insertDateQuery, "{{schema}}", h.DbSchema, 1)
	fmt.Println("Insert dates 2")

	for _, d := range incomingEvent.Dates {
		if d.VenueId != nil {
			fmt.Println("Insert dates 3")

			// Check if user can create an event in 'd.VenueId'
			venuePermissions, err := h.GetUserVenuePermissions(
				gc, tx, userId, *incomingEvent.OrganizerId, *d.VenueId)
			if err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if venuePermissions.Has(app.PermChooseVenue) {
				gc.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
				return
			}
			fmt.Println("Insert dates 4")

			if d.SpaceId != nil {
				spaceOK, err := h.IsSpaceInVenue(gc, tx, *d.SpaceId, *d.VenueId)
				if err != nil {
					gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				if !spaceOK {
					gc.JSON(http.StatusInternalServerError, gin.H{"error": "invalid venue/space combination"})
					return
				}
			}
			fmt.Println("Insert dates 5")
		}

		_, err = tx.Exec(
			ctx, sql,
			eventId,
			d.VenueId,
			d.SpaceId,
			d.StartDate,
			d.StartTime,
			d.EndDate,
			d.EndTime,
			d.EntryTime,
			d.AllDay,
			userId)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event_date: %v", err)})
			return
		}
		fmt.Println("Insert dates 6")
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

	// TODO: languages
	// TODO: tags

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

	if e.hasLocation() {
		fmt.Println("Location:", e.Location)
	}
}

// Validate validates the EventDataIncoming struct
func (e *EventDataIncoming) Validate() error {
	var errs []string

	// Validate Title
	if strings.TrimSpace(e.Title) == "" {
		errs = append(errs, "title is required")
	}

	// Validate Description
	if strings.TrimSpace(e.Description) == "" {
		errs = append(errs, "description is required")
	}

	// Validate Subtitle
	if err := validateOptionalNonEmptyString("subtitle", e.Subtitle); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate TeaserText
	if err := validateOptionalNonEmptyString("teaser_text", e.TeaserText); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate Tags
	if len(e.Tags) > 0 {
		for i, tag := range e.Tags {
			if strings.TrimSpace(tag) == "" {
				errs = append(errs, fmt.Sprintf("tag at position %d is empty", i))
			}
		}
	}

	// Validate SourceUrl (optional)
	if err := validateOptionalURL("source_url", e.SourceUrl); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate OnlineEventUrl (optional)
	if err := validateOptionalURL("online_event_url", e.OnlineEventUrl); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate OrganizerId (permissons to use this will be checked in the actual queries)
	if e.OrganizerId == nil {
		errs = append(errs, "organizer_id is required")
	} else if *e.OrganizerId < 0 {
		errs = append(errs, "organizer_id is invalid")
	}

	// Validate VenueId/Location
	if !e.hasVenue() && !e.hasLocation() {
		errs = append(errs, "event must have either venue_id or location")
	} else if e.hasVenue() && e.hasLocation() {
		errs = append(errs, "event cannot have both venue_id and location")
	}

	// Validate ExternalId (optional)
	if err := validateOptionalNonEmptyString("external_id", e.ExternalId); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate ParticipationInfo (optional)
	if err := validateOptionalNonEmptyString("participation_info", e.ParticipationInfo); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate MinAge and MaxAge
	if e.MinAge != nil {
		if *e.MinAge < 0 || *e.MinAge > 100 {
			errs = append(errs, "min_age must be between 0 and 100")
		}
	}
	if e.MaxAge != nil {
		if *e.MaxAge < 0 || *e.MaxAge > 100 {
			errs = append(errs, "max_age must be between 0 and 100")
		}
	}
	// If both are present, check that max_age >= min_age
	if e.MinAge != nil && e.MaxAge != nil {
		if *e.MaxAge < *e.MinAge {
			errs = append(errs, "max_age cannot be less than min_age")
		}
	}

	// Validate Languages
	if len(e.Languages) > 0 {
		for i, lang := range e.Languages {
			trimmed := strings.TrimSpace(lang)
			if len(trimmed) != 2 {
				errs = append(errs, fmt.Sprintf("languages[%d] must be an ISO 639-1 code (2 letters)", i))
			}
		}
	}

	// Validate MeetingPoint (optional)
	if err := validateOptionalNonEmptyString("meeting_point", e.MeetingPoint); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate MaxAttendees (optional)
	if e.MaxAttendees != nil {
		if *e.MaxAttendees <= 0 {
			errs = append(errs, "max_attendees must be greater than 0 if provided")
		}
	}

	// Validate CurrencyCode (optional)
	if e.CurrencyCode != nil {
		trimmed := strings.TrimSpace(*e.CurrencyCode)
		if trimmed != "" && len(trimmed) != 3 {
			errs = append(errs, "currency_code must be a 3-letter ISO 4217 code if provided")
		}
	}

	// Validate Custom (optional)
	if err := validateOptionalNonEmptyString("custom", e.Custom); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate Style (optional)
	if err := validateOptionalNonEmptyString("style", e.Style); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate ReleaseStatusId (optional)
	if e.ReleaseStatusId != nil {
		if *e.ReleaseStatusId < 1 || *e.ReleaseStatusId > 5 {
			errs = append(errs, "release_status_id must be between 1 and 5 if provided")
		}
	}

	// Validate ReleaseDate (optional)
	if err := validateOptionalDate("release_date", e.ReleaseDate); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate Dates
	if len(e.Dates) == 0 {
		errs = append(errs, "at least one date is required")
	} else {
		for i, date := range e.Dates {
			// Required fields
			if strings.TrimSpace(date.StartDate) == "" {
				errs = append(errs, fmt.Sprintf("dates[%d].start_date is required", i))
			} else if err := validateOptionalDate(fmt.Sprintf("dates[%d].start_date", i), &date.StartDate); err != nil {
				errs = append(errs, err.Error())
			}

			if strings.TrimSpace(date.StartTime) == "" {
				errs = append(errs, fmt.Sprintf("dates[%d].start_time is required", i))
			} else if err := validateOptionalTime(fmt.Sprintf("dates[%d].start_time", i), &date.StartTime); err != nil {
				errs = append(errs, err.Error())
			}

			// Optional fields
			if err := validateOptionalDate(fmt.Sprintf("dates[%d].end_date", i), date.EndDate); err != nil {
				errs = append(errs, err.Error())
			}
			if err := validateOptionalTime(fmt.Sprintf("dates[%d].end_time", i), date.EndTime); err != nil {
				errs = append(errs, err.Error())
			}
			if err := validateOptionalTime(fmt.Sprintf("dates[%d].entry_time", i), date.EntryTime); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}

	/*
			TODO:
			- Validate location
			- Validate startDate/startTime in the past
		    - Validate endDate/endTime before startDate/startTime
			- OrganizerId, check permissions for organizer_id
			- Dates, check permissions for using: venue_id, space_id
			- OccasionTypeId, check if valid value
			- PriceTypeId, check if valid value
	*/

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func (e *EventDataIncoming) hasVenue() bool {
	return e.VenueId != nil
}

func (e *EventDataIncoming) hasLocation() bool {
	return e.Location != nil
}

// validateOptionalNonEmptyString checks if an optional string pointer is non-empty.
// - If value is nil, it's considered valid.
// - If value is non-nil but empty or whitespace-only, it returns an error.
func validateOptionalNonEmptyString(fieldName string, value *string) error {
	if value != nil && strings.TrimSpace(*value) == "" {
		return fmt.Errorf("%s cannot be empty if provided", fieldName)
	}
	return nil
}

// validateOptionalURL validates a pointer to a string as a URL.
// - If the pointer is nil, it's considered valid.
// - If the string is empty or whitespace, it's considered invalid.
// - Otherwise, it checks for valid URL format and http/https scheme.
func validateOptionalURL(fieldName string, value *string) error {
	if value == nil {
		// No value provided → valid
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		// Value is provided but empty → error
		return fmt.Errorf("%s cannot be empty if provided", fieldName)
	}

	// Validate URL format
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s must be a valid URL", fieldName)
	}

	// Ensure it starts with http:// or https://
	if !(strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://")) {
		return fmt.Errorf("%s must start with http:// or https://", fieldName)
	}

	return nil
}

// ValidateOptionalDate validates an optional date string in the format YYYY-MM-DD.
// - If the pointer is nil or empty, it is considered valid.
// - Otherwise, it checks if the value matches the format "2006-01-02".
func validateOptionalDate(fieldName string, value *string) error {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return fmt.Errorf("%s must be in format YYYY-MM-DD", fieldName)
	}
	return nil
}

// validateOptionalTime validates an optional time string in the format HH:MM (24-hour).
// - If the pointer is nil or empty, it is considered valid.
// - Otherwise, it checks if the value matches the format "15:04".
func validateOptionalTime(fieldName string, value *string) error {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	if _, err := time.Parse("15:04", trimmed); err != nil {
		return fmt.Errorf("%s must be in format HH:MM (24-hour)", fieldName)
	}
	return nil
}
