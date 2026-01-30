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
	Version                             string
	APIName                             string
	APIVersion                          string
	MainDbPool                          *pgxpool.Pool
	Config                              Config
	SqlGetOrganization                  string
	SqlGetEvent                         string
	SqlGetEventDates                    string
	SqlGetEventsProjected               string
	SqlGetEventsTypeSummary             string
	SqlGetUserOrganizationPermissions   string
	SqlGetUserEffectiveVenuePermissions string
	SqlGetAdminOrganization             string
	SqlInsertOrganization               string
	SqlUpdateOrganization               string
	SqlGetAdminVenue                    string
	SqlInsertVenue                      string
	SqlUpdateVenue                      string
	SqlAdminGetSpace                    string
	SqlInsertSpace                      string
	SqlUpdateSpace                      string
	SqlAdminGetEvent                    string
	SqlAdminGetEventTypes               string
	SqlAdminGetEventImages              string
	SqlAdminGetEventLinks               string
	SqlAdminGetEventDates               string
	SqlAdminInsertEventDate             string
	SqlAdminUpdateEventDate             string
	SqlTypeGenreLookup                  string
	SqlChoosableOrganizationVenues      string
	SqlChoosableVenueSpaces             string
	SqlChoosableEventTypes              string
	SqlChoosableEventGenres             string
	SqlGetGeojsonVenues                 string
	SqlAdminGetOrganizationDashboard    string
	SqlAdminGetOrganizationVenues       string
	SqlAdminGetOrganizationEvents       string
	SqlAdminGetOrganizationMemberLink   string
	SqlAdminGetOrganizationMembers      string
	SqlAdminGetPermissionList           string
	SqlQueryUserOrgEventsOverview       string
	SqlAdminGetUserEventNotifications   string
	SqlAdminChoosableOrganizations      string
	SqlAdminGetChoosableUserEventPlaces string
	SqlAdminEvent                       string
	SqlAdminSpacesForEvent              string
	SqlInsertPlutoImage                 string
	SqlUpdatePlutoImageMeta             string
	JwtKey                              []byte `json:"jwt_secret"`
}

var UranusInstance *Uranus

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

	UranusInstance = &uranus // Optional: assign if everything succeeded
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

func (q *SqlQueryItem) LoadSql(schema string, baseApiUrl string) error {
	// Read main SQL file
	fileContent, err := os.ReadFile(q.filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(fileContent)

	// Replace {{schema}} and {{base_api_url}} placeholder
	resultStr := strings.ReplaceAll(contentStr, "{{schema}}", schema)
	resultStr = strings.ReplaceAll(resultStr, "{{base_api_url}}", baseApiUrl)

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
		{"sql/get-event.sql", &app.SqlGetEvent, nil},
		{"sql/get-event-dates.sql", &app.SqlGetEventDates, nil},
		{"sql/get-events-projected.sql", &app.SqlGetEventsProjected, nil},

		{"sql/get-events-type-summary.sql", &app.SqlGetEventsTypeSummary, nil},

		{"sql/choosable-event-genres.sql", &app.SqlChoosableEventGenres, nil},
		{"sql/get-geojson-venues.sql", &app.SqlGetGeojsonVenues, nil},

		{"sql/type-genre-lookup.sql", &app.SqlTypeGenreLookup, nil},
		{"sql/choosable-organization-venues.sql", &app.SqlChoosableOrganizationVenues, nil},

		{"sql/choosable-organization-venues.sql", &app.SqlChoosableOrganizationVenues, nil},
		{"sql/choosable-venue-spaces.sql", &app.SqlChoosableVenueSpaces, nil},

		{"sql/get-organization.sql", &app.SqlGetOrganization, nil},

		// Admin

		{"sql/admin-get-user-organization-permissions.sql", &app.SqlGetUserOrganizationPermissions, nil},
		{"sql/admin-get-user-effective-venue-permissions.sql", &app.SqlGetUserEffectiveVenuePermissions, nil},

		{"sql/admin-get-organization.sql", &app.SqlGetAdminOrganization, nil},
		{"sql/admin-insert-organization.sql", &app.SqlInsertOrganization, nil},
		{"sql/admin-update-organization.sql", &app.SqlUpdateOrganization, nil},

		{"sql/admin-get-venue.sql", &app.SqlGetAdminVenue, nil},
		{"sql/admin-insert-venue.sql", &app.SqlInsertVenue, nil},
		{"sql/admin-update-venue.sql", &app.SqlUpdateVenue, nil},

		{"sql/admin-get-space.sql", &app.SqlAdminGetSpace, nil},
		{"sql/admin-insert-space.sql", &app.SqlInsertSpace, nil},
		{"sql/admin-update-space.sql", &app.SqlUpdateSpace, nil},

		{"sql/admin-get-event.sql", &app.SqlAdminGetEvent, nil},
		{"sql/admin-get-event-types.sql", &app.SqlAdminGetEventTypes, nil},
		{"sql/admin-get-event-images.sql", &app.SqlAdminGetEventImages, nil},
		{"sql/admin-get-event-links.sql", &app.SqlAdminGetEventLinks, nil},
		{"sql/admin-get-event-dates.sql", &app.SqlAdminGetEventDates, nil},

		{"sql/admin-update-event-date.sql", &app.SqlAdminUpdateEventDate, nil},
		{"sql/admin-insert-event-date.sql", &app.SqlAdminInsertEventDate, nil},

		{"sql/admin-get-user-event-notifications.sql", &app.SqlAdminGetUserEventNotifications, nil},

		{"sql/admin-user-spaces-for-event.sql", &app.SqlAdminSpacesForEvent, nil},

		{"sql/admin-choosable-organizations.sql", &app.SqlAdminChoosableOrganizations, nil},
		{"sql/admin-get-choosable-user-event-places.sql", &app.SqlAdminGetChoosableUserEventPlaces, nil},

		{"sql/admin-get-organization-dashboard.sql", &app.SqlAdminGetOrganizationDashboard, nil},
		{"sql/admin-get-organization-events.sql", &app.SqlAdminGetOrganizationEvents, nil},

		{"sql/admin-get-organization-member-link.sql", &app.SqlAdminGetOrganizationMemberLink, nil},
		{"sql/admin-get-organization-members.sql", &app.SqlAdminGetOrganizationMembers, nil},
		{"sql/admin-get-permission-list.sql", &app.SqlAdminGetPermissionList, nil},

		{"sql/admin-get-organization-venues.sql", &app.SqlAdminGetOrganizationVenues, nil},

		{"sql/insert-pluto-image.sql", &app.SqlInsertPlutoImage, nil},
		{"sql/update-pluto-image-meta.sql", &app.SqlUpdatePlutoImageMeta, nil},
	}

	for i := range queries {
		q := &queries[i] // pointer to slice element
		if err := q.LoadSql(app.Config.DbSchema, app.Config.BaseApiUrl); err != nil {
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
