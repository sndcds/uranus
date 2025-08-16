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

type OrganizerQueries struct {
	OrganizerQuery string
}

var _gOrganizerQueries OrganizerQueries
var _gOrganizerOnce sync.Once

type Organizer struct {
	Id                 int            `json:"id"`
	HoldingOrganizerId sql.NullInt64  `json:"holding_organizer_id"`
	Name               string         `json:"name"`
	Description        sql.NullString `json:"description"`
	AddressAddition    sql.NullString `json:"address_addition"`
	Street             sql.NullString `json:"street"`
	HouseNumber        sql.NullString `json:"house_number"`
	PostalCode         sql.NullString `json:"postal_code"`
	City               sql.NullString `json:"city"`
	StateCode          sql.NullString `json:"state_code"`
	CountryCode        sql.NullString `json:"country_code"`
	ContactEmail       sql.NullString `json:"contact_email"`
	ContactPhone       sql.NullString `json:"contact_phone"`
	WebsiteUrl         sql.NullString `json:"website_url"`
	LegalFormId        sql.NullInt64  `json:"legal_form_id"`
	NonProfit          sql.NullBool   `json:"nonprofit"`
	Created            time.Time      `json:"created_at"`
	LastModified       sql.NullTime   `json:"modified_at"`
	CreatorUserId      sql.NullInt64  `json:"created_by"`
	ModifierUserId     sql.NullInt64  `json:"modified_by"`

	// Properties not in the database
	CanEditFlag              bool
	VenuesCount              int
	SpacesCount              int
	CurrentActiveEventsCount int
}

func _organizerInit() {
	_gOrganizerOnce.Do(func() {
		_organizerQueryTemplate :=
			`SELECT
    			id,
    			holding_organizer_id,
				name,
				description,
				address_addition,
				street,
				house_number,
				postal_code,
				city,
				state_code,
				country_code,
				contact_email,
				contact_phone,
				website_url,
				legal_form_id,
				nonprofit,
				created_at,
				modified_at,
				created_by,
				modified_by    
			FROM {{schema}}.organizer WHERE id = $1`

		_gOrganizerQueries.OrganizerQuery = strings.ReplaceAll(_organizerQueryTemplate, "{{schema}}", app.Singleton.Config.DbSchema)
	})
}

func NewOrganizer() Organizer {
	return Organizer{
		VenuesCount:              -1,
		SpacesCount:              -1,
		CurrentActiveEventsCount: -1,
	}
}

func (organizer Organizer) Print() {
	fmt.Println("Organizer:")
	fmt.Println("  id:", organizer.Id)
	fmt.Println("  holding_organizer_id:", organizer.HoldingOrganizerId)
	fmt.Println("  name:", organizer.Name)
	fmt.Println("  description:", app.SqlNullStringToString(organizer.Description))
	fmt.Println("  address_addition:", app.SqlNullStringToString(organizer.AddressAddition))
	fmt.Println("  street:", app.SqlNullStringToString(organizer.Street))
	fmt.Println("  house_number:", app.SqlNullStringToString(organizer.HouseNumber))
	fmt.Println("  postal_code:", app.SqlNullStringToString(organizer.PostalCode))
	fmt.Println("  city:", app.SqlNullStringToString(organizer.City))
	fmt.Println("  state_code:", app.SqlNullStringToString(organizer.StateCode))
	fmt.Println("  country_code:", app.SqlNullStringToString(organizer.CountryCode))
	fmt.Println("  contact_email:", app.SqlNullStringToString(organizer.ContactEmail))
	fmt.Println("  contact_phone:", app.SqlNullStringToString(organizer.ContactPhone))
	fmt.Println("  website_url:", app.SqlNullStringToString(organizer.WebsiteUrl))
	fmt.Println("  legal_form_id:", app.SqlNullInt64ToString(organizer.LegalFormId))
	fmt.Println("  nonprofit:", app.SqlNullBoolToString(organizer.NonProfit))
	fmt.Println("  created_at:", organizer.Created.Format(time.RFC3339))
	fmt.Println("  modified_at:", app.SqlNullTimeToString(organizer.LastModified))
	fmt.Println("  created_by:", app.SqlNullInt64ToString(organizer.CreatorUserId))
	fmt.Println("  modified_by:", app.SqlNullInt64ToString(organizer.ModifierUserId))
}

func (organizer Organizer) CompactString() string {
	return fmt.Sprintf("%d: %s in %s %s",
		organizer.Id,
		organizer.Name,
		app.SqlNullStringToString(organizer.PostalCode),
		app.SqlNullStringToString(organizer.City),
	)
}

func GetOrganizerById(app app.Uranus, organizerId int) (Organizer, int) {
	_organizerInit()

	// Execute the query to get the organizer
	rows, err := app.MainDbPool.Query(context.Background(), _gOrganizerQueries.OrganizerQuery, organizerId)
	if err != nil {
		fmt.Println(err)
		// If an error occurs, convert it to an HTTP error code
		httpErr := app.DbErrorToHTTP(err)
		return Organizer{}, httpErr // Return an empty organizer struct and the error code
	}
	defer rows.Close()

	// Prepare a variable to hold the organizer result
	var organizer Organizer

	// Loop through the rows (though we expect just one row)
	if rows.Next() {
		// Scan the row into the organizer struct
		err := rows.Scan(
			&organizer.Id,
			&organizer.HoldingOrganizerId,
			&organizer.Name,
			&organizer.Description,
			&organizer.AddressAddition,
			&organizer.Street,
			&organizer.HouseNumber,
			&organizer.PostalCode,
			&organizer.City,
			&organizer.StateCode,
			&organizer.CountryCode,
			&organizer.ContactEmail,
			&organizer.ContactPhone,
			&organizer.WebsiteUrl,
			&organizer.LegalFormId,
			&organizer.NonProfit,
			&organizer.Created,
			&organizer.LastModified,
			&organizer.CreatorUserId,
			&organizer.ModifierUserId,
		)
		if err != nil {
			fmt.Println(err)
			// If there is an error scanning, return a 500 internal server error
			return Organizer{}, http.StatusInternalServerError
		}
	} else {
		// If no rows were found, return 404 Not Found
		return Organizer{}, http.StatusNotFound
	}

	// Return the single organizer and a 200 OK status
	return organizer, http.StatusOK
}

func GetOrganizersByUserId(app app.Uranus, ctx *gin.Context, userId int) ([]Organizer, error) {
	_organizerInit()
	query := `
		SELECT
			uol.organizer_id AS organizer_id,
			uo.name AS organizer_name,
			ur.organizer AS can_edit
		FROM app.user_organizer_links AS uol
		JOIN app.user AS u ON u.id = uol.user_id
		JOIN app.organizer AS uo ON uo.id = uol.organizer_id
		JOIN app.user_role AS ur ON ur.id = uol.user_role_id
		WHERE uol.user_id = $1
		ORDER BY
		organizer_name
	`
	rows, err := app.MainDbPool.Query(context.Background(), query, userId)
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var organizers []Organizer

	for rows.Next() {
		var organizer = NewOrganizer()
		err := rows.Scan(&organizer.Id, &organizer.Name, &organizer.CanEditFlag)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read organizer data"})
			return nil, fmt.Errorf("failed to read organizer data: %w", err)
		}

		organizers = append(organizers, organizer)
	}

	// Check for any errors during row iteration
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating rows"})
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Return the slice of organizers
	return organizers, nil
}

func (organizer *Organizer) GetStats(app app.Uranus, ctx *gin.Context) error {

	query := `
	SELECT
		COUNT(DISTINCT v.id) AS count_venues,
		COUNT(DISTINCT s.id) AS count_spaces,
		COALESCE(COUNT(DISTINCT e.id), 0) AS count_events
	FROM app.organizer o
	JOIN app.venue v ON v.organizer_id = o.id
	LEFT JOIN app.space s ON s.venue_id = v.id
	LEFT JOIN app.event e ON e.venue_id = v.id
	LEFT JOIN app.event_date ed ON ed.event_id = e.id
	WHERE o.id = $1
	AND (ed.start >= NOW() OR ed.start IS NULL)`

	rows, err := app.MainDbPool.Query(context.Background(), query, organizer.Id)
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&organizer.VenuesCount, &organizer.SpacesCount, &organizer.CurrentActiveEventsCount)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read organizer stats"})
			return fmt.Errorf("failed to read organizer stats: %w", err)
		}
	}

	return nil
}
