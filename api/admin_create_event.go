package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// TODO: Review code

type incomingLocation struct {
	Name        *string `json:"name"`
	Description *string `json:"description" binding:"required"`
	Street      string  `json:"street" binding:"required"`
	HouseNumber *string `json:"house_number"`
	PostalCode  string  `json:"postal_code" binding:"required"`
	City        string  `json:"city" binding:"required"`
	Country     string  `json:"country" binding:"required"`
	State       *string `json:"state"`
	Lon         float64 `json:"lon"`
	Lat         float64 `json:"lat"`
}

type incomingTypeGenrePair struct {
	TypeId  int  `json:"type_id" binding:"required"`
	GenreId *int `json:"genre_id"`
}

type incomingEventDate struct {
	StartDate string  `json:"start_date" binding:"required"`
	StartTime string  `json:"start_time" binding:"required"`
	EndDate   *string `json:"end_date"`
	EndTime   *string `json:"end_time"`
	EntryTime *string `json:"entry_time"`
	VenueId   *int    `json:"venue_id"`
	SpaceId   *int    `json:"space_id"`
	AllDay    *bool   `json:"all_day"`
}

type incomingEvent struct {
	Title             string          `json:"title" binding:"required"`
	Description       string          `json:"description" binding:"required"`
	Subtitle          *string         `json:"subtitle"`
	Summary           *string         `json:"summary"`
	Tags              []string        `json:"tags"`
	SourceUrl         *string         `json:"source_url"`
	OnlineLink        *string         `json:"online_link"`
	OrganizationId    *int            `json:"organization_id" binding:"required"`
	VenueId           *int            `json:"venue_id"`
	SpaceId           *int            `json:"space_id"`
	ExternalId        *string         `json:"external_id"`
	ParticipationInfo *string         `json:"participation_info"`
	OccasionTypeId    *int            `json:"occasion_type_id"`
	Languages         []string        `json:"languages"`
	MinAge            *int            `json:"min_age"`
	MaxAge            *int            `json:"max_age"`
	MeetingPoint      *string         `json:"meeting_point"`
	MaxAttendees      *int            `json:"max_attendees"`
	PriceType         model.PriceType `json:"price_type"`
	Currency          *string         `json:"currency"`
	TicketFlags       []string        `json:"ticket_flags"`
	Custom            *string         `json:"custom"`
	Style             *string         `json:"style"`
	ReleaseStatus     *string         `json:"release_status"`
	ReleaseDate       *string         `json:"release_date"`

	Location       *incomingLocation       `json:"location"`
	Dates          []incomingEventDate     `json:"dates" binding:"required"`
	TypeGenrePairs []incomingTypeGenrePair `json:"types"`
}

func (h *ApiHandler) AdminCreateEvent(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	// Read the body
	body, err := io.ReadAll(gc.Request.Body)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	if len(body) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	// Decode JSON with unknown field rejection
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var payload incomingEvent
	if err := decoder.Decode(&payload); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxErr):
			gc.JSON(
				http.StatusBadRequest,
				gin.H{"error": fmt.Sprintf("invalid JSON syntax at offset %d", syntaxErr.Offset)},
			)
		case errors.As(err, &typeErr):
			field := typeErr.Field
			if field == "" {
				field = "(unknown)"
			}
			gc.JSON(
				http.StatusBadRequest,
				gin.H{"error": fmt.Sprintf("invalid type for field %q", field)},
			)
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	if decoder.More() {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "multiple JSON objects are not allowed"})
		return
	}

	// payload is now safe to use, valid JSON, correct types, no unknown fields, safe to use

	err = json.Unmarshal(body, &payload)
	if err != nil {
		var unmarshalTypeError *json.UnmarshalTypeError
		var syntaxErr *json.SyntaxError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxErr):
			gc.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid JSON syntax (at offset %d)", syntaxErr.Offset),
			})
			return
		case errors.As(err, &unmarshalTypeError):
			field := unmarshalTypeError.Field
			if field == "" {
				field = "(unknown field)"
			}
			gc.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("invalid type for field %q: expected %v but got %v", field, unmarshalTypeError.Type, unmarshalTypeError.Value),
				"hint":  "check numeric and boolean values — don't quote numbers or booleans",
			})
			return
		case errors.Is(err, io.EOF):
			gc.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
			return
		case errors.As(err, &invalidUnmarshalError):
			gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		default:
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Validation
	validationErr := payload.Validate()
	if validationErr != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
		return
	}

	if payload.OrganizationId == nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "organizationId is required"})
		return
	}

	var newEventId int

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		txErr := h.CheckOrganizationAllPermissions(
			gc, tx, userId, *payload.OrganizationId, app.PermChooseAsEventOrganization|app.PermAddEvent)
		if txErr != nil {
			return txErr
		}

		// Check if user can create an event in 'incomingEvent.VenueId'
		if payload.VenueId != nil {
			venuePermissions, err := h.GetUserEffectiveVenuePermissions(gc, tx, userId, *payload.VenueId)
			if err != nil {
				return &ApiTxError{Code: http.StatusForbidden, Err: err}
			}
			if !venuePermissions.Has(app.PermChooseVenue) {
				return ApiErrForbidden("")
			}
		}

		// Insert event Information
		insertEventQuery := `
		INSERT INTO {{schema}}.event (
			organization_id,
			venue_id,
			space_id,
			external_id,
		  	release_status,
			title,
			subtitle,
			description,
			summary,
		  	languages,
			created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`
		query := strings.Replace(insertEventQuery, "{{schema}}", h.Config.DbSchema, 1)

		err = tx.QueryRow(
			ctx, query,
			payload.OrganizationId,
			payload.VenueId,
			payload.SpaceId,
			payload.ExternalId,
			payload.ReleaseStatus,
			payload.Title,
			payload.Subtitle,
			payload.Description,
			payload.Summary,
			payload.Languages,
			userId,
		).Scan(&newEventId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusForbidden,
				Err:  fmt.Errorf("failed to insert event: %v, userId: %d", err, userId),
			}
		}

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
		query = strings.Replace(insertDateQuery, "{{schema}}", h.DbSchema, 1)

		for _, d := range payload.Dates {
			if d.VenueId != nil {
				// Check if user can create an event in 'd.VenueId'
				venuePermissions, err := h.GetUserEffectiveVenuePermissions(gc, tx, userId, *d.VenueId)
				if err != nil {
					return &ApiTxError{Code: http.StatusInternalServerError, Err: err}
				}
				if !venuePermissions.Has(app.PermChooseVenue) {
					return ApiErrForbidden("")
				}

				if d.SpaceId != nil {
					spaceOK, err := h.IsSpaceInVenue(gc, tx, *d.SpaceId, *d.VenueId)
					if err != nil {
						return &ApiTxError{Code: http.StatusInternalServerError, Err: err}
					}
					if !spaceOK {
						return &ApiTxError{
							Code: http.StatusInternalServerError,
							Err:  fmt.Errorf("invalid venue/space combination"),
						}
					}
				}
			}

			_, err = tx.Exec(
				ctx, query,
				newEventId,
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
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to insert event_date: %v", err),
				}
			}
		}

		// Insert Type + Genre pairs
		queryTemplate := `
		INSERT INTO {{schema}}.event_type_link (event_id, type_id, genre_id)
		VALUES ($1, $2, $3)`
		query = strings.Replace(queryTemplate, "{{schema}}", h.Config.DbSchema, 1)

		for _, pair := range payload.TypeGenrePairs {
			_, err := tx.Exec(ctx, query, newEventId, pair.TypeId, pair.GenreId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to insert type-genre pair: %v", err),
				}
			}
		}

		// TODO: Insert languages
		// TODO: Insert tags

		err = RefreshEventProjections(ctx, tx, "event", []int{newEventId})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %v", err),
			}
		}

		return nil
	})
	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	gc.JSON(http.StatusCreated, gin.H{"event_id": newEventId})
}

// Validate validates the incomingEvent struct
func (e *incomingEvent) Validate() error {
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
	err := app.ValidateOptionalNonEmptyString("subtitle", e.Subtitle)
	if err != nil {
		errs = append(errs, err.Error())
	}

	// Validate Summary
	err = app.ValidateOptionalNonEmptyString("summary", e.Summary)
	if err != nil {
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
	if err := app.ValidateOptionalUrl("source_url", e.SourceUrl); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate OnlineLink (optional)
	if err := app.ValidateOptionalUrl("online_link", e.OnlineLink); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate OrganizationId (permissons to use this will be checked in the actual sql)
	if e.OrganizationId == nil {
		errs = append(errs, "organization_id is required")
	} else if *e.OrganizationId < 0 {
		errs = append(errs, "organization_id is invalid")
	}

	// Validate VenueId
	if !e.hasVenue() {
		errs = append(errs, "event must have venueId")
	}

	// Validate ExternalId (optional)
	if err := app.ValidateOptionalNonEmptyString("external_id", e.ExternalId); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate ParticipationInfo (optional)
	if err := app.ValidateOptionalNonEmptyString("participation_info", e.ParticipationInfo); err != nil {
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
	if err := app.ValidateOptionalNonEmptyString("meeting_point", e.MeetingPoint); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate MaxAttendees (optional)
	if e.MaxAttendees != nil {
		if *e.MaxAttendees <= 0 {
			errs = append(errs, "max_attendees must be greater than 0 if provided")
		}
	}

	// Validate Currency(optional)
	if e.Currency != nil {
		trimmed := strings.TrimSpace(*e.Currency)
		if trimmed != "" && len(trimmed) != 3 {
			errs = append(errs, "currency must be a 3-letter ISO 4217 code if provided")
		}
	}

	// Validate Custom (optional)
	if err := app.ValidateOptionalNonEmptyString("custom", e.Custom); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate Style (optional)
	if err := app.ValidateOptionalNonEmptyString("style", e.Style); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate ReleaseStatus (optional)
	if e.ReleaseStatus != nil {
		ok, _ := IsEventReleaseStatus("release_status", e.ReleaseStatus)
		if !ok {
			errs = append(errs, "unknown release_status")
		}
	} else {
		defaultValue := "draft"
		e.ReleaseStatus = &defaultValue
	}

	// Validate ReleaseDate (optional)
	if err := app.ValidateOptionalDate("release_date", e.ReleaseDate); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate Dates
	if len(e.Dates) == 0 {
		errs = append(errs, "at least one date is required")
	} else {
		for i, date := range e.Dates {
			// start_date
			if strings.TrimSpace(date.StartDate) == "" {
				errs = append(errs, fmt.Sprintf("dates[%d].start_date is required", i))
			} else if err := app.ValidateOptionalDate(fmt.Sprintf("dates[%d].start_date", i), &date.StartDate); err != nil {
				errs = append(errs, err.Error())
			} else {
				beforeToday, err := isBeforeToday(date.StartDate)
				if err != nil {
					errs = append(errs, fmt.Sprintf(
						"dates[%d].start_date has invalid format", i,
					))
				} else if beforeToday {
					errs = append(errs, fmt.Sprintf(
						"dates[%d].start_date must not be in the past", i,
					))
				}
			}

			// Optional fields
			if err := app.ValidateOptionalDate(fmt.Sprintf("dates[%d].end_date", i), date.EndDate); err != nil {
				errs = append(errs, err.Error())
			}
			if err := app.ValidateOptionalTime(fmt.Sprintf("dates[%d].end_time", i), date.EndTime); err != nil {
				errs = append(errs, err.Error())
			}
			if err := app.ValidateOptionalTime(fmt.Sprintf("dates[%d].entry_time", i), date.EntryTime); err != nil {
				errs = append(errs, err.Error())
			}

			errs = append(errs, validateEventDate(date, i)...)
		}
	}

	/*
		TODO:
		- Validate location
		- Validate startDate/startTime in the past
		- Validate endDate/endTime before startDate/startTime
		- OrganizationId, check permissions for organization_id
		- Dates, check permissions for using: venue_id, space_id
		- OccasionType, check if valid value
		- PriceType, check if valid value
	*/

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

func (e *incomingEvent) hasVenue() bool {
	return e.VenueId != nil
}

func isBeforeToday(dateStr string) (bool, error) {
	// Parse YYYY-MM-DD
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false, err
	}

	now := time.Now()
	today := time.Date(
		now.Year(), now.Month(), now.Day(),
		0, 0, 0, 0,
		now.Location(),
	)

	return d.Before(today), nil
}

func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

func parseTime(timeStr string) (time.Time, error) {
	return time.Parse("15:04", timeStr)
}

func validateEventDate(e incomingEventDate, index int) []string {
	var errs []string

	// Parse start date
	startDate, err := time.Parse("2006-01-02", e.StartDate)
	if err != nil {
		return []string{
			fmt.Sprintf("dates[%d].start_date has invalid format (expected YYYY-MM-DD)", index),
		}
	}

	// Parse start time
	startTime, err := time.Parse("15:04", e.StartTime)
	if err != nil {
		return []string{
			fmt.Sprintf("dates[%d].start_time has invalid format (expected HH:MM)", index),
		}
	}

	// Rule: start_date >= today
	now := time.Now()
	today := time.Date(
		now.Year(), now.Month(), now.Day(),
		0, 0, 0, 0,
		now.Location(),
	)

	if startDate.Before(today) {
		errs = append(errs,
			fmt.Sprintf("dates[%d].start_date must be today or in the future", index),
		)
	}

	// Parse optional end_date
	var endDate time.Time
	hasEndDate := false

	if e.EndDate != nil {
		endDate, err = time.Parse("2006-01-02", *e.EndDate)
		if err != nil {
			errs = append(errs,
				fmt.Sprintf("dates[%d].end_date has invalid format (expected YYYY-MM-DD)", index),
			)
		} else {
			hasEndDate = true
		}
	}

	// Parse optional end_time
	var endTime time.Time
	hasEndTime := false

	if e.EndTime != nil {
		endTime, err = time.Parse("15:04", *e.EndTime)
		if err != nil {
			errs = append(errs,
				fmt.Sprintf("dates[%d].end_time has invalid format (expected HH:MM)", index),
			)
		} else {
			hasEndTime = true
		}
	}

	// Rule: end_date > start_date
	if hasEndDate && !endDate.After(startDate) {
		errs = append(errs,
			fmt.Sprintf("dates[%d].end_date must be after start_date", index),
		)
	}

	// Rule: end_time validation
	if hasEndTime {
		// If no end_date, assume same day as start_date
		compareDate := startDate
		if hasEndDate {
			compareDate = endDate
		}

		// Same-day check → end_time must be after start_time
		if compareDate.Equal(startDate) && !endTime.After(startTime) {
			errs = append(errs,
				fmt.Sprintf("dates[%d].end_time must be after start_time", index),
			)
		}
	}

	// Rule: entry_time < start_time
	if e.EntryTime != nil {
		entryTime, err := time.Parse("15:04", *e.EntryTime)
		if err != nil {
			errs = append(errs,
				fmt.Sprintf("dates[%d].entry_time has invalid format (expected HH:MM)", index),
			)
		} else if !entryTime.Before(startTime) {
			errs = append(errs,
				fmt.Sprintf("dates[%d].entry_time must be before start_time", index),
			)
		}
	}

	return errs
}
