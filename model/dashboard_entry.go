package model

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DashboardEntryQueries struct {
	DashboardEntryQuery string
}

var _gDashboardEntryQueries DashboardEntryQueries
var _gDashboardEntryOnce sync.Once

type DashboardEntry struct {
	OrganizerId          int            `json:"organizer_id"`
	OrganizerName        string         `json:"organizer_name"`
	OrganizerStreet      sql.NullString `json:"organizer_street"`
	OrganizerHouseNumber sql.NullString `json:"organizer_house_number"`
	OrganizerPostalCode  sql.NullString `json:"organizer_post_code"`
	OrganizerCity        sql.NullString `json:"organizer_city"`
	OrganizerWebsiteUrl  sql.NullString `json:"website_url"`
	CanEditOrganizer     bool           `json:"can_edit_organizer"`
	CanDeleteOrganizer   bool           `json:"can_delete_organizer"`
	CanAddVenue          bool           `json:"can_add_venue"`
	CanEditVenue         bool           `json:"can_edit_venue"`
	CanDeleteVenue       bool           `json:"can_delete_venue"`
	CanAddSpace          bool           `json:"can_add_space"`
	CanEditSpace         bool           `json:"can_edit_space"`
	CanDeleteSpace       bool           `json:"can_delete_space"`
	CanAddEvent          bool           `json:"can_add_event"`
	CanEditEvent         bool           `json:"can_edit_event"`
	CanDeleteEvent       bool           `json:"can_delete_event"`
	CanReleaseEvent      bool           `json:"can_release_event"`
	CanViewEventInsights bool           `json:"can_view_event_insights"`
	VenueCount           int            `json:"venue_count"`
	SpaceCount           int            `json:"space_count"`
	UpcomingEventCount   int            `json:"upcoming_event_count"`
}

type DashboardEntryJSON struct {
	OrganizerId          int     `json:"organizer_id"`
	OrganizerName        string  `json:"organizer_name"`
	OrganizerStreet      *string `json:"organizer_street"`
	OrganizerHouseNumber *string `json:"organizer_house_number"`
	OrganizerPostalCode  *string `json:"organizer_post_code"`
	OrganizerCity        *string `json:"organizer_city"`
	OrganizerWebsiteUrl  *string `json:"website_url"`
	CanEditOrganizer     bool    `json:"can_edit_organizer"`
	CanDeleteOrganizer   bool    `json:"can_delete_organizer"`
	CanAddVenue          bool    `json:"can_add_venue"`
	CanEditVenue         bool    `json:"can_edit_venue"`
	CanDeleteVenue       bool    `json:"can_delete_venue"`
	CanAddSpace          bool    `json:"can_add_space"`
	CanEditSpace         bool    `json:"can_edit_space"`
	CanDeleteSpace       bool    `json:"can_delete_space"`
	CanAddEvent          bool    `json:"can_add_event"`
	CanEditEvent         bool    `json:"can_edit_event"`
	CanDeleteEvent       bool    `json:"can_delete_event"`
	CanReleaseEvent      bool    `json:"can_release_event"`
	CanViewEventInsights bool    `json:"can_view_event_insights"`
	VenueCount           int     `json:"venue_count"`
	SpaceCount           int     `json:"space_count"`
	UpcomingEventCount   int     `json:"upcoming_event_count"`
}

func DashBoardEntryToJSON(e DashboardEntry) DashboardEntryJSON {
	return DashboardEntryJSON{
		OrganizerId:          e.OrganizerId,
		OrganizerName:        e.OrganizerName,
		OrganizerStreet:      nullToPtr(e.OrganizerStreet),
		OrganizerHouseNumber: nullToPtr(e.OrganizerHouseNumber),
		OrganizerPostalCode:  nullToPtr(e.OrganizerPostalCode),
		OrganizerCity:        nullToPtr(e.OrganizerCity),
		OrganizerWebsiteUrl:  nullToPtr(e.OrganizerWebsiteUrl),
		CanEditOrganizer:     e.CanEditOrganizer,
		CanDeleteOrganizer:   e.CanDeleteOrganizer,
		CanAddVenue:          e.CanAddVenue,
		CanEditVenue:         e.CanEditVenue,
		CanDeleteVenue:       e.CanDeleteVenue,
		CanAddSpace:          e.CanAddSpace,
		CanEditSpace:         e.CanEditSpace,
		CanDeleteSpace:       e.CanDeleteSpace,
		CanAddEvent:          e.CanAddEvent,
		CanEditEvent:         e.CanEditEvent,
		CanDeleteEvent:       e.CanDeleteEvent,
		CanReleaseEvent:      e.CanReleaseEvent,
		CanViewEventInsights: e.CanViewEventInsights,
		VenueCount:           e.VenueCount,
		SpaceCount:           e.SpaceCount,
		UpcomingEventCount:   e.UpcomingEventCount,
	}
}

func nullToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func _dashboardEntryInit() {
	_gDashboardEntryOnce.Do(func() {
		absPath, _ := filepath.Abs("queries/dashboardEntries.sql")
		fmt.Println("absPath:", absPath)
		sqlFileContent, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Println("Error reading SQL file:", err)
			return
		}

		_gDashboardEntryQueries.DashboardEntryQuery = strings.ReplaceAll(string(sqlFileContent), "{{schema}}", app.Singleton.Config.DbSchema)
		fmt.Println("Dashboard Entry Queries loaded:", _gDashboardEntryQueries)
	})
}

func NewDashboardEntry() DashboardEntry {
	return DashboardEntry{}
}

// Implement the PlaceholderReplacer interface for DashboardEntry
func (entry DashboardEntry) GetValueForKey(key string) string {
	// Define an array with 5 strings
	profileImgUrls := []string{
		"static/uploads/3.jpg",
		"static/uploads/7.png",
		"static/uploads/36abddd6-06a8-11f0-9029-1bdd58a3b2d3.webp",
		"static/uploads/143665e2-06a7-11f0-a3d4-17276fe804d0.webp",
		"static/uploads/b3603ee0-06a7-11f0-9ab9-bb76629222b2.webp"}

	switch key {
	case ".OrganizerId":
		return strconv.Itoa(entry.OrganizerId)
	case ".Name":
		return entry.OrganizerName
	case ".Address":
		return entry.GetFullAddress()
	case ".WebsiteUrl":
		return entry.OrganizerWebsiteUrl.String
	case ".Venues":
		return strconv.Itoa(entry.VenueCount)
	case ".Spaces":
		return strconv.Itoa(entry.SpaceCount)
	case ".Events":
		return strconv.Itoa(entry.UpcomingEventCount)
	case ".ProfileImgUrl":
		rand.Seed(time.Now().UnixNano())
		return profileImgUrls[rand.Intn(len(profileImgUrls))]
	default:
		return ""
	}
}

// Function to get full address
func (entry *DashboardEntry) GetFullAddress() string {
	var fullAddress string

	// Check if OrganizerStreet is valid
	if entry.OrganizerStreet.Valid {
		fullAddress += entry.OrganizerStreet.String + " "
	}

	// Add OrganizerId, making sure it's always added as a string
	fullAddress += strconv.Itoa(entry.OrganizerId) + ", "

	// Check if OrganizerPostalCode is valid
	if entry.OrganizerPostalCode.Valid {
		fullAddress += entry.OrganizerPostalCode.String + " "
	}

	// Check if OrganizerCity is valid
	if entry.OrganizerCity.Valid {
		fullAddress += entry.OrganizerCity.String
	}

	// Return the complete address
	return fullAddress
}

func (entry DashboardEntry) Print() {
	fmt.Println("DashboardEntry:")
	fmt.Println("  organizer_id:", entry.OrganizerId, "name:", entry.OrganizerName)
	fmt.Println("  organizer_street:", entry.OrganizerStreet, "organizer_house_number:", entry.OrganizerHouseNumber)
	fmt.Println("  organizer_postal_code:", entry.OrganizerPostalCode, "organizer_house_city:", entry.OrganizerCity)
	fmt.Println("  organizer edit:", entry.CanEditOrganizer, "delete:", entry.CanDeleteOrganizer)
	fmt.Println("  venue add:", entry.CanAddVenue, "edit:", entry.CanEditVenue, "delete:", entry.CanDeleteVenue)
	fmt.Println("  space add:", entry.CanAddSpace, "edit:", entry.CanEditSpace, "delete:", entry.CanDeleteSpace)
	fmt.Println("  event add:", entry.CanAddEvent, "edit:", entry.CanEditEvent, "delete:", entry.CanDeleteEvent, "release:", entry.CanReleaseEvent)
	fmt.Println("  can_view_event_insights:", entry.CanViewEventInsights)
	fmt.Println("  venue_count:", entry.VenueCount)
	fmt.Println("  space_count:", entry.SpaceCount)
	fmt.Println("  upcoming_event_count:", entry.UpcomingEventCount)
}

func GetDashboardEntriesByUserId(app app.Uranus, gc *gin.Context, userId int) ([]DashboardEntry, int) {
	_dashboardEntryInit()
	var entries []DashboardEntry
	rows, err := app.MainDbPool.Query(context.Background(), _gDashboardEntryQueries.DashboardEntryQuery, userId)
	if err != nil {
		fmt.Println(err)
		httpErr := app.DbErrorToHTTP(err)
		return entries, httpErr
	}
	defer rows.Close()

	var entry DashboardEntry
	for rows.Next() {
		err := rows.Scan(
			&entry.OrganizerId,
			&entry.OrganizerName,
			&entry.OrganizerStreet,
			&entry.OrganizerHouseNumber,
			&entry.OrganizerPostalCode,
			&entry.OrganizerCity,
			&entry.OrganizerWebsiteUrl,
			&entry.CanEditOrganizer,
			&entry.CanDeleteOrganizer,
			&entry.CanAddVenue,
			&entry.CanEditVenue,
			&entry.CanDeleteVenue,
			&entry.CanAddSpace,
			&entry.CanEditSpace,
			&entry.CanDeleteSpace,
			&entry.CanAddEvent,
			&entry.CanEditEvent,
			&entry.CanDeleteEvent,
			&entry.CanReleaseEvent,
			&entry.CanViewEventInsights,
			&entry.VenueCount,
			&entry.SpaceCount,
			&entry.UpcomingEventCount,
		)
		if err != nil {
			fmt.Println(err)
			return entries, http.StatusInternalServerError
		}
		entries = append(entries, entry)
		entry.Print()
	}

	if len(entries) == 0 {
		return entries, http.StatusNotFound
	}

	return entries, http.StatusOK
}
