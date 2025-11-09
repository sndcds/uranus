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
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5" // PostgreSQL driver
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Uranus struct {
	Version                              string
	APIName                              string
	APIVersion                           string
	MainDbPool                           *pgxpool.Pool
	Config                               Config
	SqlGetEvent                          string
	SqlGetEventsBasic                    string
	SqlGetEventsExtended                 string
	SqlGetEventsGeometry                 string
	SqlGetEventsDetailed                 string
	SqlGetEventsPerType                  string
	SqlGetAdminOrganizer                 string
	SqlUpdateOrganizer                   string
	SqlGetAdminVenue                     string
	SqlUpdateVenue                       string
	SqlGetAdminSpace                     string
	SqlUpdateSpace                       string
	SqlGetAdminEvent                     string
	SqlQueryOrganizerRoles               string
	SqlQueryVenueRoles                   string
	SqlQuerySpaceRoles                   string
	SqlQueryEventRoles                   string
	SqlChoosableOrganizerVenues          string
	SqlChoosableVenueSpaces              string
	SqlChoosableEventTypes               string
	SqlChoosableEventGenres              string
	SqlGetMetaGenresByEventType          string
	SqlGetGeojsonVenues                  string
	SqlQueryVenueByUser                  string
	SqlQuerySpacesByVenueId              string
	SqlQueryUserVenuesById               string
	SqlAdminOrganizerDashboard           string
	SqlAdminOrganizerVenues              string
	SqlAdminOrganizerEvents              string
	SqlAdminGetOrganizerEventPermissions string
	SqlQueryUserOrgEventsOverview        string
	SqlAdminUserPermissions              string
	SqlAdminGetUserEventNotification     string
	SqlAdminChoosableOrganizers          string
	SqlAdminChoosableUserEventOrganizers string
	SqlAdminEvent                        string
	SqlAdminSpacesCanAddEvent            string
	SqlAdminSpacesForEvent               string
	SqlEventImages                       string
	JwtKey                               []byte `json:"jwt_secret"`
}

var Singleton *Uranus

func New(configFilePath string) (*Uranus, error) {
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

	uranus.Log("initialize database")
	if err := uranus.InitMainDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize main DB: %w", err)
	}

	uranus.Log("prepare sql")
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

		{"queries/get-events.sql", &app.SqlGetEventsBasic, strPtr("queries/get-events-select-basic.sql")},
		{"queries/get-events.sql", &app.SqlGetEventsExtended, strPtr("queries/get-events-select-extended.sql")},
		{"queries/get-events.sql", &app.SqlGetEventsGeometry, strPtr("queries/get-events-select-geometry.sql")},
		{"queries/get-events.sql", &app.SqlGetEventsDetailed, strPtr("queries/get-events-select-detailed.sql")},

		{"queries/get-events-per-type.sql", &app.SqlGetEventsPerType, nil},

		{"queries/organizer_roles.sql", &app.SqlQueryOrganizerRoles, nil},
		{"queries/venue_roles.sql", &app.SqlQueryVenueRoles, nil},
		{"queries/space_roles.sql", &app.SqlQuerySpaceRoles, nil},
		{"queries/event-roles.sql", &app.SqlQueryEventRoles, nil},
		{"queries/choosable-event-types.sql", &app.SqlChoosableEventTypes, nil},
		{"queries/choosable-event-genres.sql", &app.SqlChoosableEventGenres, nil},
		{"queries/get-meta-genres-by-event-type.sql", &app.SqlGetMetaGenresByEventType, nil},
		{"queries/get-geojson-venues.sql", &app.SqlGetGeojsonVenues, nil},
		{"queries/userVenues.sql", &app.SqlQueryVenueByUser, nil},
		{"queries/spacesByVenueId.sql", &app.SqlQuerySpacesByVenueId, nil},
		{"queries/userVenuesById.sql", &app.SqlQueryUserVenuesById, nil},

		{"queries/choosable-organizer-venues.sql", &app.SqlChoosableOrganizerVenues, nil},
		{"queries/choosable-venue-spaces.sql", &app.SqlChoosableVenueSpaces, nil},

		// Admin
		{"queries/admin/get-admin-organizer.sql", &app.SqlGetAdminOrganizer, nil},
		{"queries/admin/update-organizer.sql", &app.SqlUpdateOrganizer, nil},

		{"queries/admin/get-admin-venue.sql", &app.SqlGetAdminVenue, nil},
		{"queries/admin/update-venue.sql", &app.SqlUpdateVenue, nil},

		{"queries/admin/get-admin-space.sql", &app.SqlGetAdminSpace, nil},
		{"queries/admin/update-space.sql", &app.SqlUpdateSpace, nil},

		{"queries/admin/get-admin-event.sql", &app.SqlGetAdminEvent, nil},

		{"queries/admin/user-permissions.sql", &app.SqlAdminUserPermissions, nil},
		{"queries/admin/get-user-event-notification.sql", &app.SqlAdminGetUserEventNotification, nil},

		{"queries/admin/user-spaces-can-add-event.sql", &app.SqlAdminSpacesCanAddEvent, nil},
		{"queries/admin/user-spaces-for-event.sql", &app.SqlAdminSpacesForEvent, nil},

		{"queries/admin/choosable-organizers.sql", &app.SqlAdminChoosableOrganizers, nil},
		{"queries/admin/choosable-user-event-organizers.sql", &app.SqlAdminChoosableUserEventOrganizers, nil},

		{"queries/admin/organizer-dashboard.sql", &app.SqlAdminOrganizerDashboard, nil},
		{"queries/admin/organizer-events.sql", &app.SqlAdminOrganizerEvents, nil},
		{"queries/admin/get-organizer-event-permissions.sql", &app.SqlAdminGetOrganizerEventPermissions, nil},

		{"queries/admin/organizer-venues.sql", &app.SqlAdminOrganizerVenues, nil},

		// TODO
		{"queries/user-org-events-overview.sql", &app.SqlQueryUserOrgEventsOverview, nil},
		{"queries/event-images.sql", &app.SqlEventImages, nil},
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

// EncryptPassword hashes a password and returns the hashed string along with any error
func EncryptPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12) // bcrypt.DefaultCost
	if err != nil {
		return "", err // Return an empty string and the error
	}
	return string(hashedPassword), nil // Return the hashed password and nil error
}

// ComparePasswords compares a plain password with a bcrypt hash
func ComparePasswords(storedHash, password string) error {
	// Compare the plain password to the bcrypt hash
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
}

func ReadSVG(path string) string {
	svgContent, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(svgContent)
}

// TruncateAtWord truncates the string at the word boundary
func TruncateAtWord(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	words := strings.Fields(s)
	var truncated string
	for _, word := range words {
		// Add the word and a space if it doesn't exceed the max length
		if len(truncated)+len(word)+1 <= maxLength {
			if truncated == "" {
				truncated = word
			} else {
				truncated += " " + word
			}
		} else {
			break
		}
	}
	if len(truncated) < len(s) {
		truncated += " ..."
	}
	return truncated
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

func IsValidDateStr(dateStr string) bool {
	_, err := time.Parse("2006-01-02", dateStr)
	return err == nil
}

func IsValidIso639_1(languageStr string) bool {
	if languageStr != "" {
		match, _ := regexp.MatchString("^[a-z]{2}$", languageStr)
		return match
	}
	return false
}
