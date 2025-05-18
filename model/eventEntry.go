package model

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type EventEntryQueries struct {
	EventEntriesByVenueQuery     string
	EventEntriesByOrganizerQuery string
}

var _gEventEntryQueries EventEntryQueries
var _gEventEntryOnce sync.Once

type EventEntry struct {
	EventId         int            `json:"event_id"`
	EventTitle      string         `json:"event_title"`
	OrganizerId     int            `json:"organizer_id"`
	OrganizerName   string         `json:"organizer_name"`
	VenueId         int            `json:"venue_id"`
	VenueName       string         `json:"venue_name"`
	SpaceId         int            `json:"space_id"`
	SpaceName       string         `json:"space_name"`
	Start           sql.NullTime   `json:"start"`
	End             sql.NullTime   `json:"end"`
	EntryTime       sql.NullString `json:"entry_time"`
	CanEdit         sql.NullBool   `json:"can_edit"`
	CanRelease      sql.NullBool   `json:"can_release"`
	CanDelete       sql.NullBool   `json:"can_delete"`
	CanViewInsights sql.NullBool   `json:"can_view_insights"`
	HasMainImage    sql.NullBool   `json:"has_main_image"`
	ImageSourceName sql.NullString `json:"image_source_name"`
}

func (entry EventEntry) Print() {
	fmt.Println("EventEntry:")
	fmt.Println("  event_id:", entry.EventId, "title:", entry.EventTitle)
}

func _eventEntryInit() {
	_gEventEntryOnce.Do(func() {
		absPath, _ := filepath.Abs("queries/eventEntriesByVenue.sql")
		sqlFileContent, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Println("Error reading SQL file:", err)
			return
		}
		_gEventEntryQueries.EventEntriesByVenueQuery = strings.ReplaceAll(string(sqlFileContent), "{{schema}}", app.Singleton.Config.DbSchema)

		absPath, _ = filepath.Abs("queries/eventEntriesByOrganizer.sql")
		sqlFileContent, err = os.ReadFile(absPath)
		if err != nil {
			fmt.Println("Error reading SQL file:", err)
			return
		}
		_gEventEntryQueries.EventEntriesByOrganizerQuery = strings.ReplaceAll(string(sqlFileContent), "{{schema}}", app.Singleton.Config.DbSchema)
	})
}

func formatNullTime(nullTime sql.NullTime) string {
	if !nullTime.Valid {
		return "" // Return a default value if NULL
	}
	return nullTime.Time.Format("02.01.06 15:04") // Format: DD.MM.YY HH:MM
}

// Implement the PlaceholderReplacer interface for EventEntry
func (entry EventEntry) GetValueForKey(key string) string {
	switch key {
	case ".EventId":
		return strconv.Itoa(entry.EventId)
	case ".EventTitle":
		return entry.EventTitle
	case ".OrganizerId":
		return strconv.Itoa(entry.OrganizerId)
	case ".OrganizerName":
		return entry.OrganizerName
	case ".VenueId":
		return strconv.Itoa(entry.VenueId)
	case ".VenueName":
		return entry.VenueName
	case ".SpaceId":
		return strconv.Itoa(entry.SpaceId)
	case ".SpaceName":
		return entry.SpaceName
	case ".Start":
		return formatNullTime(entry.Start)
		// return app.SqlNullTimeToString(entry.Start)
	case ".End":
		return app.SqlNullTimeToString(entry.End)
	case ".EntryTime":
		return app.SqlNullStringToString(entry.EntryTime)
	case ".CanEdit":
		if entry.CanEdit.Bool {
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 -960 960 960\" width=\"24px\" fill=\"currentColor\"><path d=\"M200-200h57l391-391-57-57-391 391v57Zm-80 80v-170l528-527q12-11 26.5-17t30.5-6q16 0 31 6t26 18l55 56q12 11 17.5 26t5.5 30q0 16-5.5 30.5T817-647L290-120H120Zm640-584-56-56 56 56Zm-141 85-28-29 57 57-29-28Z\"/></svg>"
		}
		// return app.SqlNullBoolToString(entry.CanEdit)
	case ".CanRelease":
		return app.SqlNullBoolToString(entry.CanRelease)
	case ".CanDelete":
		return app.SqlNullBoolToString(entry.CanDelete)
	case ".CanViewInsights":
		return app.SqlNullBoolToString(entry.CanViewInsights)
	case ".HasMainImage":
		return app.SqlNullBoolToString(entry.HasMainImage)
	case ".ImageSourceName":
		return app.SqlNullStringToString(entry.ImageSourceName)
	case ".Image":
		return `<img src="/static/uploads/` + app.SqlNullStringToString(entry.ImageSourceName) + `" width="100px">`
	default:
		return ""
	}
	return ""
}

func scanEventEntries(rows pgx.Rows) ([]EventEntry, int) {
	var entries []EventEntry

	for rows.Next() {
		var entry EventEntry
		err := rows.Scan(
			&entry.EventId,
			&entry.EventTitle,
			&entry.OrganizerId,
			&entry.OrganizerName,
			&entry.VenueId,
			&entry.VenueName,
			&entry.SpaceId,
			&entry.SpaceName,
			&entry.Start,
			&entry.End,
			&entry.EntryTime,
			&entry.CanEdit,
			&entry.CanRelease,
			&entry.CanDelete,
			&entry.CanViewInsights,
			&entry.HasMainImage,
			&entry.ImageSourceName,
		)
		if err != nil {
			fmt.Println("Error scanning row:", err)
			return nil, http.StatusInternalServerError
		}
		entries = append(entries, entry)
		entry.Print()
	}

	if len(entries) == 0 {
		return entries, http.StatusNotFound
	}

	return entries, http.StatusOK
}

func GetEventEntriesByVenueIdAndUserId(app app.Uranus, c *gin.Context, venueId int, userId int) ([]EventEntry, int) {
	_eventEntryInit()
	var entries []EventEntry
	rows, err := app.MainDb.Query(context.Background(), _gEventEntryQueries.EventEntriesByVenueQuery, venueId, userId)
	if err != nil {
		fmt.Println(err)
		httpErr := app.DbErrorToHTTP(err)
		return entries, httpErr
	}
	defer rows.Close()

	return scanEventEntries(rows)
}

func GetEventEntriesByOrganizerAndUser(app app.Uranus, c *gin.Context, organizerId int, userId int) ([]EventEntry, int) {
	_eventEntryInit()
	var entries []EventEntry
	rows, err := app.MainDb.Query(context.Background(), _gEventEntryQueries.EventEntriesByOrganizerQuery, organizerId, userId)
	if err != nil {
		fmt.Println(err)
		httpErr := app.DbErrorToHTTP(err)
		return entries, httpErr
	}
	defer rows.Close()

	return scanEventEntries(rows)
}
