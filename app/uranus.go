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
	SqlQueryOrganizerRoles               string
	SqlQueryVenueRoles                   string
	SqlQuerySpaceRoles                   string
	SqlQueryEventRoles                   string
	SqlChoosableOrganizerVenues          string
	SqlChoosableVenueSpaces              string
	SqlChoosableEventTypes               string
	SqlChoosableEventGenres              string
	SqlGetMetaGenresByEventType          string
	SqlQueryEvent                        string
	SqlQueryVenueForMap                  string
	SqlQueryVenueByUser                  string
	SqlQuerySpacesByVenueId              string
	SqlQueryUserVenuesById               string
	SqlAdminOrganizerDashboard           string
	SqlAdminOrganizerVenues              string
	SqlAdminOrganizerEvents              string
	SqlQueryUserOrgEventsOverview        string
	SqlAdminUserPermissions              string
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

func (app *Uranus) PrepareSql() error {
	type queryItem struct {
		filePath string
		target   *string
	}

	queries := []queryItem{
		{"queries/organizer_roles.sql", &app.SqlQueryOrganizerRoles},
		{"queries/venue_roles.sql", &app.SqlQueryVenueRoles},
		{"queries/space_roles.sql", &app.SqlQuerySpaceRoles},
		{"queries/event-roles.sql", &app.SqlQueryEventRoles},
		{"queries/choosable-event-types.sql", &app.SqlChoosableEventTypes},
		{"queries/choosable-event-genres.sql", &app.SqlChoosableEventGenres},
		{"queries/get-meta-genres-by-event-type.sql", &app.SqlGetMetaGenresByEventType},
		{"queries/queryEvent.sql", &app.SqlQueryEvent},
		{"queries/queryVenueForMap.sql", &app.SqlQueryVenueForMap},
		{"queries/userVenues.sql", &app.SqlQueryVenueByUser},
		{"queries/spacesByVenueId.sql", &app.SqlQuerySpacesByVenueId},
		{"queries/userVenuesById.sql", &app.SqlQueryUserVenuesById},

		{"queries/choosable-organizer-venues.sql", &app.SqlChoosableOrganizerVenues},
		{"queries/choosable-venue-spaces.sql", &app.SqlChoosableVenueSpaces},

		{"queries/admin/organizer-dashboard.sql", &app.SqlAdminOrganizerDashboard},
		{"queries/admin/organizer-venues.sql", &app.SqlAdminOrganizerVenues},
		{"queries/admin/organizer-events.sql", &app.SqlAdminOrganizerEvents},
		{"queries/admin/choosable-organizers.sql", &app.SqlAdminChoosableOrganizers},
		{"queries/admin/choosable-user-event-organizers.sql", &app.SqlAdminChoosableUserEventOrganizers},

		{"queries/user-org-events-overview.sql", &app.SqlQueryUserOrgEventsOverview},
		{"queries/admin/admin-user-permissions.sql", &app.SqlAdminUserPermissions},
		{"queries/admin/admin-event.sql", &app.SqlAdminEvent},
		{"queries/admin/admin-user-spaces-can-add-event.sql", &app.SqlAdminSpacesCanAddEvent},
		{"queries/admin/admin-user-spaces-for-event.sql", &app.SqlAdminSpacesForEvent},
		{"queries/event-images.sql", &app.SqlEventImages},
	}

	for _, q := range queries {
		if err := loadFileReplaceAllSchema(q.filePath, app.Config.DbSchema, q.target); err != nil {
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
