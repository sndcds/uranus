package model

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"log"
	"net/http"
	"time"
)

type EventDateInfo struct {
	DateId           int
	EventId          int
	VenueId          int
	OrganizerId      int
	SpaceId          int
	SpaceTypeId      int
	Start            time.Time
	EventTitle       string
	EventDescription string
	VenueName        string
	VenuePostalCode  string
	VenueCity        string
	VenueTypes       string
	EventTypes       string
	GenreTypes       string
	OrganizerName    string
	SpaceName        string
	SpaceTypeName    string
	ImageUrl         string
	CanEdit          bool
}

func (eventDateInfo EventDateInfo) Print() {
	fmt.Println("EventDateInfo:")
	fmt.Println("  DateId:", eventDateInfo.DateId)
	fmt.Println("  EventId:", eventDateInfo.EventId)
	fmt.Println("  VenueId:", eventDateInfo.VenueId)
	fmt.Println("  OrganizerId:", eventDateInfo.OrganizerId)
	fmt.Println("  SpaceId:", eventDateInfo.SpaceId)
	fmt.Println("  SpaceTypeId:", eventDateInfo.SpaceTypeId)
	fmt.Println("  Start:", eventDateInfo.Start)
	fmt.Println("  EventTitle:", eventDateInfo.EventTitle)
	fmt.Println("  EventDescription:", eventDateInfo.EventDescription)
	fmt.Println("  VenueName:", eventDateInfo.VenueName)
	fmt.Println("  VenuePostalCode:", eventDateInfo.VenuePostalCode)
	fmt.Println("  VenueCity:", eventDateInfo.VenueCity)
	fmt.Println("  VenueTypes:", eventDateInfo.VenueTypes)
	fmt.Println("  EventTypes:", eventDateInfo.EventTypes)
	fmt.Println("  GenreTypes:", eventDateInfo.GenreTypes)
	fmt.Println("  OrganizerName:", eventDateInfo.OrganizerName)
	fmt.Println("  SpaceName:", eventDateInfo.SpaceName)
	fmt.Println("  SpaceTypeName:", eventDateInfo.SpaceTypeName)
	fmt.Println("  ImageUrl:", eventDateInfo.ImageUrl)
	fmt.Println("  CanEdit:", eventDateInfo.CanEdit)
}

func GetEventDateInfosByUserId(app app.Uranus, ctx *gin.Context, i18n_locale string, userId int) ([]EventDateInfo, error) {

	query := `
		WITH 
			FilteredI18n AS (
				SELECT id, iso_639_1
				FROM app.i18n_locale
				WHERE iso_639_1 = $1 -- Ensuring only one i18n_locale_id
			),
			VenueTypes AS (
				SELECT vt.type_id AS venue_type_id, vt.name AS venue_name
				FROM app.venue_type vt
				JOIN FilteredI18n fi ON fi.id = vt.i18n_locale_id
			),
			EventTypes AS (
				SELECT et.type_id AS event_type_id, et.name AS event_name
				FROM app.event_type et
				JOIN FilteredI18n fi ON fi.id = et.i18n_locale_id
			),
			GenreTypes AS (
				SELECT gt.type_id AS genre_type_id, gt.name AS genre_name
				FROM app.genre_type gt
				JOIN FilteredI18n fi ON fi.id = gt.i18n_locale_id
			),
			SpaceTypes AS (
				SELECT st.type_id AS space_type_id, st.name AS space_name
				FROM app.space_type st
				JOIN FilteredI18n fi ON fi.id = st.i18n_locale_id
			),
			UserPermissions AS (
				SELECT 
					e.id AS event_id,
					ed.id AS event_date_id,
					uvl.user_id,
					MIN(ed.start) AS event_start_first,
					MAX(ed.start) AS event_start_last,
					CASE 
						WHEN ur.event = TRUE THEN TRUE
						ELSE FALSE
					END AS can_edit
				FROM app.event e
				JOIN app.venue v ON v.id = e.venue_id
				JOIN app.user_venue_links uvl ON uvl.venue_id = v.id
				JOIN app.event_date ed ON ed.event_id = e.id
				JOIN app.user_role ur ON ur.id = uvl.user_role_id
				WHERE uvl.user_id = $2 AND ed.start >= NOW()
				GROUP BY e.id, ed.id, v.id, v.name, ur.event, uvl.user_id
			)
		SELECT 
			ed.id AS event_date_id,
			e.id AS event_id,
			v.id AS venue_id,
			v.name AS venue_name,
			v.postal_code AS venue_postcode,
			v.city AS venue_city,
			e.title AS event_title,
			e.description AS event_description,
			ed.start AS event_start,
			STRING_AGG(DISTINCT et.event_name, ', ') AS event_types,
			STRING_AGG(DISTINCT gt.genre_name, ', ') AS genre_types,
			o.id AS organizer_id,
			o.name AS organizer_name,
			s.id AS space_id,
			s.name AS space_name,
			st.space_name AS space_type_name,
			STRING_AGG(DISTINCT vt.venue_name, ', ') AS venue_types,
			NULLIF(CONCAT('', 'static/uploads/', i.source_name), CONCAT('', 'static/uploads/')) AS image_url,
			up.can_edit
		FROM app.event e
		JOIN app.event_date ed ON e.id = ed.event_id
		LEFT JOIN app.venue v ON v.id = COALESCE(ed.venue_id, e.venue_id)
		LEFT JOIN app.space s ON s.id = COALESCE(ed.space_id, e.space_id)
		LEFT JOIN app.event_type_links elt ON elt.event_id = e.id
		LEFT JOIN EventTypes et ON et.event_type_id = elt.event_type_id
		LEFT JOIN app.event_genre_links egl ON egl.event_id = e.id
		LEFT JOIN GenreTypes gt ON gt.genre_type_id = egl.genre_type_id
		LEFT JOIN app.venue_link_types vlt ON vlt.venue_id = v.id
		LEFT JOIN VenueTypes vt ON vt.venue_type_id = vlt.venue_type_id
		LEFT JOIN SpaceTypes st ON st.space_type_id = s.space_type_id
		LEFT JOIN app.organizer o ON o.id = e.organizer_id
		LEFT JOIN app.event_date_image_links edli ON edli.event_date_id = ed.id AND edli.main_image = TRUE
		LEFT JOIN app.event_image_links eli ON eli.event_id = e.id AND eli.main_image = TRUE
		LEFT JOIN app.Image i ON i.id = COALESCE(edli.image_id, eli.image_id)
		INNER JOIN UserPermissions up ON up.event_date_id = ed.id
		WHERE 
			ed.start >= NOW() AND
			ed.start <= '2027-01-01' AND
			v.city = 'Flensburg'
		GROUP BY 
			ed.id, e.id, v.id, e.title, e.description, o.name, o.id, v.name, v.postal_code, v.city, ed.start, e.created_at, s.id, s.name, st.space_name, i.source_name, up.user_id, up.can_edit
		ORDER BY ed.start`

	rows, err := app.MainDb.Query(context.Background(), query, i18n_locale, userId)
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var eventDateInfos []EventDateInfo

	for rows.Next() {
		var eventDateInfo = EventDateInfo{}
		var eventTitle sql.NullString
		var eventDescription sql.NullString
		var venueName sql.NullString
		var venuePostalCode sql.NullString
		var venueCity sql.NullString
		var venueTypes sql.NullString
		var eventTypes sql.NullString
		var genreTypes sql.NullString
		var organizerName sql.NullString
		var spaceName sql.NullString
		var spaceTypeName sql.NullString
		var imageUrl sql.NullString

		err := rows.Scan(
			&eventDateInfo.DateId,
			&eventDateInfo.EventId,
			&eventDateInfo.VenueId,
			&venueName,
			&venuePostalCode,
			&venueCity,
			&eventTitle,
			&eventDescription,
			&eventDateInfo.Start,
			&eventTypes,
			&genreTypes,
			&eventDateInfo.OrganizerId,
			&organizerName,
			&eventDateInfo.SpaceId,
			&spaceName,
			&spaceTypeName,
			&venueTypes,
			&imageUrl,
			&eventDateInfo.CanEdit,
		)

		eventDateInfo.EventTitle = eventTitle.String
		eventDateInfo.EventDescription = eventDescription.String
		eventDateInfo.VenueName = venueName.String
		eventDateInfo.VenuePostalCode = venuePostalCode.String
		eventDateInfo.VenueCity = venueCity.String
		eventDateInfo.VenueTypes = venueTypes.String
		eventDateInfo.EventTypes = eventTypes.String
		eventDateInfo.GenreTypes = genreTypes.String
		eventDateInfo.OrganizerName = organizerName.String
		eventDateInfo.SpaceName = spaceName.String
		eventDateInfo.SpaceTypeName = spaceTypeName.String
		eventDateInfo.ImageUrl = imageUrl.String

		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read organizer data"})
			return nil, fmt.Errorf("failed to read event data: %w", err)
		}
		eventDateInfos = append(eventDateInfos, eventDateInfo)
	}

	// Check for any errors during row iteration
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating rows"})
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Return the slice of organizers
	return eventDateInfos, nil
}

func GetEventDateInfosByUserId2(app app.Uranus, ctx *gin.Context, userId int) ([]EventDateInfo, error) {

	query := `
		SELECT
			ed.id AS event_date_id,
			e.id AS event_id,
			e.title AS event_title,
			v.name AS event_venue_name,
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

	rows, err := app.MainDb.Query(context.Background(), query, userId)
	if err != nil {
		log.Printf("Query failed: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var eventDateInfos []EventDateInfo

	for rows.Next() {
		var eventDateInfo = EventDateInfo{}
		var eventDateStartLast time.Time

		err := rows.Scan(
			&eventDateInfo.DateId,
			&eventDateInfo.EventId,
			&eventDateInfo.EventTitle,
			&eventDateInfo.VenueName,
			&eventDateInfo.Start,
			&eventDateStartLast,
			&eventDateInfo.CanEdit)
		if err != nil {
			log.Printf("Failed to scan row: %v\n", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read organizer data"})
			return nil, fmt.Errorf("failed to read event data: %w", err)
		}
		eventDateInfos = append(eventDateInfos, eventDateInfo)
	}

	// Check for any errors during row iteration
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating rows"})
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Return the slice of organizers
	return eventDateInfos, nil
}
