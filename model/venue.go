package model

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type VenueQueries struct {
	VenueQuery string
}

var _gVenueQueries VenueQueries
var _gVenueOnce sync.Once

type Venue struct {
	Id             int            `json:"id"`
	OrganizerId    int            `json:"organizer_id"`
	Name           string         `json:"name"`
	Description    sql.NullString `json:"description"`
	Street         sql.NullString `json:"street"`
	HouseNumber    sql.NullString `json:"house_number"`
	PostalCode     sql.NullString `json:"postal_code"`
	City           sql.NullString `json:"city"`
	StateCode      sql.NullString `json:"state_code"`
	CountryCode    sql.NullString `json:"country_code"`
	ContactEmail   sql.NullString `json:"contact_email"`
	ContactPhone   sql.NullString `json:"contact_phone"`
	OpenedAt       sql.NullTime   `json:"opened_at"`
	ClosedAt       sql.NullTime   `json:"closed_at"`
	Created        time.Time      `json:"created_at"`
	LastModified   sql.NullTime   `json:"modified_at"`
	CreatorUserId  sql.NullInt64  `json:"created_by"`
	ModifierUserId sql.NullInt64  `json:"modified_by"`

	// Properties not in the database
	CanEditVenue             bool
	CanEditSpace             bool
	CanEditEvent             bool
	SpacesCount              int
	CurrentActiveEventsCount int
}

func _venueInit() {
	_gVenueOnce.Do(func() {
		_venueQueryTemplate :=
			`SELECT
    			id,
    			organizer_id,
				name,
				description,
				street,
				house_number,
				postal_code,
				city,
				state_code,
				country_code,
				contact_email,
				contact_phone,
				opened_at,
				closed_at,
				created_at,
				modified_at,
				created_by,
				modified_by    
			FROM {{schema}}.venue WHERE id = $1`
		_gVenueQueries.VenueQuery = strings.ReplaceAll(_venueQueryTemplate, "{{schema}}", app.Singleton.Config.DbSchema)
	})
}

func NewVenue() Venue {
	return Venue{}
}

func (venue Venue) Print() {
	fmt.Println("Venue:")
	fmt.Println("  id:", venue.Id)
	fmt.Println("  organizer_id:", venue.OrganizerId)
	fmt.Println("  name:", venue.Name)
	fmt.Println("  description:", app.SqlNullStringToString(venue.Description))
	fmt.Println("  street:", app.SqlNullStringToString(venue.Street))
	fmt.Println("  house_number:", app.SqlNullStringToString(venue.HouseNumber))
	fmt.Println("  postal_code:", app.SqlNullStringToString(venue.PostalCode))
	fmt.Println("  city:", app.SqlNullStringToString(venue.City))
	fmt.Println("  state_code:", app.SqlNullStringToString(venue.StateCode))
	fmt.Println("  country_code:", app.SqlNullStringToString(venue.CountryCode))
	fmt.Println("  contact_email:", app.SqlNullStringToString(venue.ContactEmail))
	fmt.Println("  contact_phone:", app.SqlNullStringToString(venue.ContactPhone))
	fmt.Println("  opened_at:", app.SqlNullTimeToString(venue.OpenedAt))
	fmt.Println("  closed_at:", app.SqlNullTimeToString(venue.ClosedAt))
	fmt.Println("  created_at:", venue.Created.Format(time.RFC3339))
	fmt.Println("  modified_at:", app.SqlNullTimeToString(venue.LastModified))
	fmt.Println("  created_by:", app.SqlNullInt64ToString(venue.CreatorUserId))
	fmt.Println("  modified_by:", app.SqlNullInt64ToString(venue.ModifierUserId))
}

func (venue Venue) CompactString() string {
	return fmt.Sprintf("%d: %s in %s %s",
		venue.Id,
		venue.Name,
		app.SqlNullStringToString(venue.PostalCode),
		app.SqlNullStringToString(venue.City),
	)
}

func GetVenueById(app app.Uranus, venueId int) (Venue, int) {
	_venueInit()
	// Execute the query to get the venue
	rows, err := app.MainDb.Query(context.Background(), _gVenueQueries.VenueQuery, venueId)
	if err != nil {
		fmt.Println(err)
		// If an error occurs, convert it to an HTTP error code
		httpErr := app.DbErrorToHTTP(err)
		return Venue{}, httpErr // Return an empty venue struct and the error code
	}
	defer rows.Close()

	// Prepare a venue variable to hold the result
	var venue Venue

	// Loop through the rows (though we expect just one row)
	if rows.Next() {
		// Scan the row into the venue struct
		err := rows.Scan(
			&venue.Id,
			&venue.OrganizerId,
			&venue.Name,
			&venue.Description,
			&venue.Street,
			&venue.HouseNumber,
			&venue.PostalCode,
			&venue.City,
			&venue.StateCode,
			&venue.CountryCode,
			&venue.ContactEmail,
			&venue.ContactPhone,
			&venue.OpenedAt,
			&venue.ClosedAt,
			&venue.Created,
			&venue.LastModified,
			&venue.CreatorUserId,
			&venue.ModifierUserId,
		)
		if err != nil {
			fmt.Println(err)
			// If there is an error scanning, return a 500 internal server error
			return Venue{}, http.StatusInternalServerError
		}
	} else {
		// If no rows were found, return 404 Not Found
		return Venue{}, http.StatusNotFound
	}

	// Return the single venue and a 200 OK status
	return venue, http.StatusOK
}

func GetVenuesByUserId(app app.Uranus, ctx *gin.Context, userId int) ([]Venue, error) {

	query := `
		SELECT
			uvl.venue_id AS venue_id,
			v.name AS venue_name,
			ur.venue AS can_edit_venue,
			ur.space AS can_edit_space,
			ur.event AS can_edit_event
		FROM
			app.user_venue_links uvl
		JOIN
			app.user u ON u.id = uvl.user_id
		JOIN
			app.venue v ON v.id = uvl.venue_id
		JOIN
			app.user_role ur ON ur.id = uvl.user_role_id
		WHERE
			uvl.user_id = $1
		ORDER BY
    		v.name`

	rows, err := app.MainDb.Query(context.Background(), query, userId)
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var venues []Venue

	for rows.Next() {
		var venue = NewVenue()
		err := rows.Scan(&venue.Id, &venue.Name, &venue.CanEditVenue, &venue.CanEditSpace, &venue.CanEditEvent)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read organizer data"})
			return nil, fmt.Errorf("failed to read organizer data: %w", err)
		}

		venues = append(venues, venue)
	}

	// Check for any errors during row iteration
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating rows"})
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Return the slice of organizers
	return venues, nil
}

func (venue *Venue) GetStats(app app.Uranus, ctx *gin.Context) error {

	query := `
		SELECT
			COUNT(DISTINCT s.id) AS count_spaces,
			COUNT(DISTINCT e.id) AS count_events
		FROM
			app.venue v
		JOIN
			app.space s ON s.venue_id = v.id
		LEFT JOIN
			app.event e ON e.venue_id = v.id
		LEFT JOIN
			app.event_date ed ON ed.event_id = e.id
		WHERE
			v.id = $1
		AND (ed.start >= NOW() OR ed.start IS NULL)`

	rows, err := app.MainDb.Query(context.Background(), query, venue.Id)
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&venue.SpacesCount, &venue.CurrentActiveEventsCount)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read organizer stats"})
			return fmt.Errorf("failed to read organizer stats: %w", err)
		}
	}

	return nil
}
