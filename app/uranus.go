package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5" // PostgreSQL driver
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO: Review code

type Uranus struct {
	Version                                string
	APIName                                string
	APIVersion                             string
	MainDbPool                             *pgxpool.Pool
	Config                                 Config
	SqlGetOrganizer                        string
	SqlGetEvent                            string
	SqlGetEventDates                       string
	SqlGetEventsBasic                      string
	SqlGetEventsExtended                   string
	SqlGetEventsDetailed                   string
	SqlGetEventsTypeSummary                string
	SqlGetUserOrganizerPermissions         string
	SqlGetUserVenuePermissions             string
	SqlGetAdminOrganizer                   string
	SqlUpdateOrganizer                     string
	SqlGetAdminVenue                       string
	SqlUpdateVenue                         string
	SqlGetAdminSpace                       string
	SqlUpdateSpace                         string
	SqlGetAdminEvent                       string
	SqlGetAdminEventDates                  string
	SqlGetAdminEventTypes                  string
	SqlChoosableOrganizerVenues            string
	SqlChoosableVenueSpaces                string
	SqlChoosableEventTypes                 string
	SqlChoosableEventGenres                string
	SqlGetGeojsonVenues                    string
	SqlAdminGetOrganizerDashboard          string
	SqlAdminGetOrganizerVenues             string
	SqlAdminGetOrganizerEvents             string
	SqlAdminGetOrganizerAddEventPermission string
	SqlAdminGetPermissionList              string
	SqlQueryUserOrgEventsOverview          string
	SqlAdminUserPermissions                string
	SqlAdminGetUserEventNotification       string
	SqlAdminChoosableOrganizers            string
	SqlAdminChoosableUserEventOrganizers   string
	SqlAdminChoosableUserEventVenues       string
	SqlAdminChoosableUserVenuesSpaces      string
	SqlAdminEvent                          string
	SqlAdminSpacesForEvent                 string
	JwtKey                                 []byte `json:"jwt_secret"`
}

var Singleton *Uranus

func Initialize(configFilePath string) (*Uranus, error) {
	var uranus Uranus

	uranus.Version = "1.0.0"
	uranus.APIName = "Uranus"
	uranus.APIVersion = "1.0.0"

	uranus.Log("load configuration")
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&uranus.Config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Check configuration
	uranus.Config.ProfileImageQuality = ClampFloat32(uranus.Config.ProfileImageQuality, 30, 100)

	if uranus.Config.PlutoImageMaxFileSize == 0 {
		uranus.Config.PlutoImageMaxFileSize = 10 * 1014 * 1024 // 10 Mb
	}

	if uranus.Config.PlutoImageMaxPx == 0 {
		uranus.Config.PlutoImageMaxPx = 1920
	}

	uranus.Log("initialize database")
	if err := uranus.InitMainDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize main DB: %w", err)
	}

	uranus.Log("prepare sql_utils")
	if err := uranus.PrepareSql(); err != nil {
		return nil, fmt.Errorf("failed to prepare SQL statements: %w", err)
	}

	Singleton = &uranus // Optional: assign if everything succeeded
	uranus.Log("succesfully initialized")

	uranus.Log("connect to Pluto image service")

	return &uranus, nil
}

func (app *Uranus) Log(msg string) {
	if app.Config.Verbose {
		fmt.Println("app:", msg)
	}
}

func (app *Uranus) LoadConfig(fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &app.Config)
	if err != nil {
		return err
	}

	// Set default if not specified in the JSON config
	if app.Config.AuthTokenExpirationTime == 0 {
		app.Config.AuthTokenExpirationTime = 600 // default: 10 minutes
	}

	app.Config.Print()
	return nil
}

type SqlQueryItem struct {
	filePath              string
	target                *string
	modeDependentFilePath *string
}

func (q *SqlQueryItem) LoadSql(schema string) error {
	// Read main SQL file
	fileContent, err := os.ReadFile(q.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(fileContent)

	// Replace {{schema}} placeholder
	resultStr := strings.ReplaceAll(contentStr, "{{schema}}", schema)

	// Handle modeDependentFilePath if present
	if q.modeDependentFilePath != nil && *q.modeDependentFilePath != "" {
		modeContent, err := os.ReadFile(*q.modeDependentFilePath)
		if err != nil {
			return fmt.Errorf("failed to read mode-dependent file: %w", err)
		}

		resultStr = strings.ReplaceAll(resultStr, "{{mode-dependent-select}}", string(modeContent))
	}

	// Store final SQL in target
	if q.target != nil {
		*q.target = resultStr
	} else {
		return fmt.Errorf("target pointer is nil")
	}

	return nil
}

func strPtr(s string) *string {
	return &s
}

func (app *Uranus) PrepareSql() error {
	queries := []SqlQueryItem{
		// Public
		{"queries/get-event.sql", &app.SqlGetEvent, nil},
		{"queries/get-event-dates.sql", &app.SqlGetEventDates, nil},

		{"queries/get-events.sql", &app.SqlGetEventsBasic, strPtr("queries/get-events-select-basic.sql")},
		{"queries/get-events.sql", &app.SqlGetEventsExtended, strPtr("queries/get-events-select-extended.sql")},
		{"queries/get-events.sql", &app.SqlGetEventsDetailed, strPtr("queries/get-events-select-detailed.sql")},
		{"queries/get-events-type-summary.sql", &app.SqlGetEventsTypeSummary, nil},

		{"queries/choosable-event-types.sql", &app.SqlChoosableEventTypes, nil},
		{"queries/choosable-event-genres.sql", &app.SqlChoosableEventGenres, nil},
		{"queries/get-geojson-venues.sql", &app.SqlGetGeojsonVenues, nil},

		{"queries/choosable-organizer-venues.sql", &app.SqlChoosableOrganizerVenues, nil},
		{"queries/choosable-venue-spaces.sql", &app.SqlChoosableVenueSpaces, nil},

		{"queries/get-organizer.sql", &app.SqlGetOrganizer, nil},

		// Admin
		{"queries/admin-get-user-organizer-permissions.sql", &app.SqlGetUserOrganizerPermissions, nil},
		{"queries/admin-get-user-venue-permissions.sql", &app.SqlGetUserVenuePermissions, nil},

		{"queries/admin-get-organizer.sql", &app.SqlGetAdminOrganizer, nil},
		{"queries/admin-update-organizer.sql", &app.SqlUpdateOrganizer, nil},

		{"queries/admin-get-venue.sql", &app.SqlGetAdminVenue, nil},
		{"queries/admin-update-venue.sql", &app.SqlUpdateVenue, nil},

		{"queries/admin-get-space.sql", &app.SqlGetAdminSpace, nil},
		{"queries/admin-update-space.sql", &app.SqlUpdateSpace, nil},

		{"queries/admin-get-event.sql", &app.SqlGetAdminEvent, nil},
		{"queries/admin-get-event-dates.sql", &app.SqlGetAdminEventDates, nil},
		{"queries/admin-get-event-types.sql", &app.SqlGetAdminEventTypes, nil},

		{"queries/admin-user-permissions.sql", &app.SqlAdminUserPermissions, nil},
		{"queries/admin-get-user-event-notification.sql", &app.SqlAdminGetUserEventNotification, nil},

		{"queries/admin-user-spaces-for-event.sql", &app.SqlAdminSpacesForEvent, nil},

		{"queries/admin-choosable-organizers.sql", &app.SqlAdminChoosableOrganizers, nil},
		{"queries/admin-choosable-user-event-organizers.sql", &app.SqlAdminChoosableUserEventOrganizers, nil},
		{"queries/admin-choosable-user-event-venues.sql", &app.SqlAdminChoosableUserEventVenues, nil},
		{"queries/admin-choosable-user-venues-spaces.sql", &app.SqlAdminChoosableUserVenuesSpaces, nil},

		{"queries/admin-organizer-dashboard.sql", &app.SqlAdminGetOrganizerDashboard, nil},
		{"queries/admin-get-organizer-events.sql", &app.SqlAdminGetOrganizerEvents, nil},
		{"queries/admin-get-organizer-add-event-permission.sql", &app.SqlAdminGetOrganizerAddEventPermission, nil},

		{"queries/admin-get-permission-list.sql", &app.SqlAdminGetPermissionList, nil},

		{"queries/admin-organizer-venues.sql", &app.SqlAdminGetOrganizerVenues, nil},
	}

	for i := range queries {
		q := &queries[i] // pointer to slice element
		if err := q.LoadSql(app.Config.DbSchema); err != nil {
			return err
		}
	}

	return nil
}

func (app *Uranus) InitMainDB() error {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", app.Config.DbUser, app.Config.DbPassword, app.Config.DbHost, app.Config.DbPort, app.Config.DbName)

	var err error
	app.MainDbPool, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
		return err
	}

	app.Log("database successfully initialized")

	return nil
}

func (app *Uranus) CloseAllDBs() {
	if app.MainDbPool != nil {
		app.MainDbPool.Close()
	}
}

func ReadSVG(path string) string {
	svgContent, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(svgContent)
}

// Function to convert database errors to HTTP status codes
func (app *Uranus) DbErrorToHTTP(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Check for "no rows" error (record not found)
	if errors.Is(err, pgx.ErrNoRows) {
		return http.StatusNotFound
	}

	// Check for PostgreSQL-specific errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // Unique constraint violation
			return http.StatusConflict
		case "23503": // Foreign key violation
			return http.StatusBadRequest
		case "42P01": // Undefined table
			return http.StatusInternalServerError
		default:
			return http.StatusInternalServerError
		}
	}

	// Default to 500 Internal Server Error
	return http.StatusInternalServerError
}

// Utility function to extract string from sql.NullString
func SqlNullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "NULL" // Return empty string if NULL
}

// Utility function to extract time from sql.NullTime
func SqlNullTimeToString(nt sql.NullTime) string {
	if nt.Valid {
		return nt.Time.Format(time.RFC3339)
	}
	return "NULL" // Return empty string if NULL
}

// Utility function to extract int64 from sql.NullInt64
func SqlNullInt64ToInt(n sql.NullInt64) int64 {
	if n.Valid {
		return n.Int64
	}
	return 0 // Return 0 if NULL (you can choose another default value)
}

// Utility function to extract string from sql.NullInt64
func SqlNullInt64ToString(n sql.NullInt64) string {
	if n.Valid {
		return fmt.Sprintf("%d", n.Int64)
	}
	return "NULL" // Return "NULL" if NULL
}

// Utility function to extract bool from sql.NullBool
func SqlNullBoolToBool(nb sql.NullBool) bool {
	if nb.Valid {
		return nb.Bool
	}
	return false // Return false if NULL (you can choose another default value)
}

// Utility function to extract string from sql.NullBool
func SqlNullBoolToString(nb sql.NullBool) string {
	if nb.Valid {
		return fmt.Sprintf("%t", nb.Bool)
	}
	return "NULL" // Return "NULL" if NULL
}
