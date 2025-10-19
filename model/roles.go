package model

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
	"net/http"
)

type Permissions struct {
	EditOrganizer     bool
	DeleteOrganizer   bool
	AddVenue          bool
	EditVenue         bool
	DeleteVenue       bool
	AddSpace          bool
	EditSpace         bool
	DeleteSpace       bool
	AddEvent          bool
	EditEvent         bool
	DeleteEvent       bool
	ReleaseEvent      bool
	ViewEventInsights bool
}

type RoleMetadata struct {
	UserRoleID int
	RoleName   string
	Permissions
}

type OrganizerRole struct {
	OrganizerID   int
	OrganizerName string
	RoleMetadata
}

type VenueRole struct {
	VenueID       int
	VenueName     string
	OrganizerID   int
	OrganizerName string
	RoleMetadata
}

type SpaceRole struct {
	SpaceID       int
	SpaceName     string
	VenueID       int
	VenueName     string
	OrganizerID   int
	OrganizerName string
	RoleMetadata
}

type EventRole struct {
	EventID            int
	EventTitle         string
	EventOrganizerID   int
	EventOrganizerName string
	SpaceID            int
	SpaceName          string
	VenueID            int
	VenueName          string
	OrganizerID        int
	OrganizerName      string
	UserRoleID         int
	RoleName           string
	Permissions        Permissions
}

type AllRoles struct {
	OrganizerRoles map[int]OrganizerRole `json:"organizer_roles"`
	VenueRoles     map[int]VenueRole     `json:"venue_roles"`
	SpaceRoles     map[int]SpaceRole     `json:"space_roles"`
	EventRoles     map[int]EventRole     `json:"event_roles"`
}

func permissionScanTargets(p *Permissions) []interface{} {
	return []interface{}{
		&p.EditOrganizer,
		&p.DeleteOrganizer,
		&p.AddVenue,
		&p.EditVenue,
		&p.DeleteVenue,
		&p.AddSpace,
		&p.EditSpace,
		&p.DeleteSpace,
		&p.AddEvent,
		&p.EditEvent,
		&p.DeleteEvent,
		&p.ReleaseEvent,
		&p.ViewEventInsights,
	}
}

func mergePermissions(a, b Permissions) Permissions {
	return Permissions{
		EditOrganizer:     a.EditOrganizer || b.EditOrganizer,
		DeleteOrganizer:   a.DeleteOrganizer || b.DeleteOrganizer,
		AddVenue:          a.AddVenue || b.AddVenue,
		EditVenue:         a.EditVenue || b.EditVenue,
		DeleteVenue:       a.DeleteVenue || b.DeleteVenue,
		AddSpace:          a.AddSpace || b.AddSpace,
		EditSpace:         a.EditSpace || b.EditSpace,
		DeleteSpace:       a.DeleteSpace || b.DeleteSpace,
		AddEvent:          a.AddEvent || b.AddEvent,
		EditEvent:         a.EditEvent || b.EditEvent,
		DeleteEvent:       a.DeleteEvent || b.DeleteEvent,
		ReleaseEvent:      a.ReleaseEvent || b.ReleaseEvent,
		ViewEventInsights: a.ViewEventInsights || b.ViewEventInsights,
	}
}

func ensureOrganizerExists(
	organizerMap map[int]OrganizerRole,
	organizerID int,
	organizerName string,
) {
	if _, found := organizerMap[organizerID]; !found {
		organizerMap[organizerID] = OrganizerRole{
			OrganizerID:   organizerID,
			OrganizerName: organizerName,
			RoleMetadata: RoleMetadata{
				UserRoleID:  -1,
				RoleName:    "generated",
				Permissions: Permissions{
					// All permissions default to false (zero values)
				},
			},
		}
	}
}

func FetchOrganizerRoles(gc *gin.Context, dbPool *pgxpool.Pool, userID int) (map[int]OrganizerRole, error) {
	ctx := gc.Request.Context()
	rows, err := dbPool.Query(ctx, app.Singleton.SqlQueryOrganizerRoles, userID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	organizerMap := make(map[int]OrganizerRole)

	for rows.Next() {
		var role OrganizerRole

		scanTargets := []interface{}{
			&role.OrganizerID,
			&role.OrganizerName,
			&role.UserRoleID,
			&role.RoleName,
		}
		scanTargets = append(scanTargets, permissionScanTargets(&role.Permissions)...)

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Merge permissions for duplicate organizer IDs
		if existing, found := organizerMap[role.OrganizerID]; found {
			role.Permissions = mergePermissions(existing.Permissions, role.Permissions)
		}

		organizerMap[role.OrganizerID] = role
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return organizerMap, nil
}

func FetchVenueRoles(gc *gin.Context, dbPool *pgxpool.Pool, userID int) (map[int]VenueRole, error) {
	ctx := gc.Request.Context()

	rows, err := dbPool.Query(ctx, app.Singleton.SqlQueryVenueRoles, userID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	venueMap := make(map[int]VenueRole)

	for rows.Next() {
		var role VenueRole

		scanTargets := []interface{}{
			&role.VenueID,
			&role.VenueName,
			&role.OrganizerID,
			&role.OrganizerName,
			&role.UserRoleID,
			&role.RoleName,
		}
		scanTargets = append(scanTargets, permissionScanTargets(&role.Permissions)...)

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if existing, found := venueMap[role.VenueID]; found {
			role.Permissions = mergePermissions(existing.Permissions, role.Permissions)
		}

		venueMap[role.VenueID] = role
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return venueMap, nil
}

func FetchSpaceRoles(gc *gin.Context, dbPool *pgxpool.Pool, userID int) (map[int]SpaceRole, error) {
	ctx := gc.Request.Context()
	rows, err := dbPool.Query(ctx, app.Singleton.SqlQuerySpaceRoles, userID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	spaceMap := make(map[int]SpaceRole)

	for rows.Next() {
		var role SpaceRole

		scanTargets := []interface{}{
			&role.SpaceID,
			&role.SpaceName,
			&role.VenueID,
			&role.VenueName,
			&role.OrganizerID,
			&role.OrganizerName,
			&role.UserRoleID,
			&role.RoleName,
		}
		scanTargets = append(scanTargets, permissionScanTargets(&role.Permissions)...)

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if existing, found := spaceMap[role.SpaceID]; found {
			role.Permissions = mergePermissions(existing.Permissions, role.Permissions)
		}

		spaceMap[role.SpaceID] = role
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return spaceMap, nil
}

func FetchEventRoles(gc *gin.Context, dbPool *pgxpool.Pool, userID int) (map[int]EventRole, error) {
	ctx := gc.Request.Context()

	rows, err := dbPool.Query(ctx, app.Singleton.SqlQueryEventRoles, userID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	eventMap := make(map[int]EventRole)

	for rows.Next() {
		var role EventRole

		scanTargets := []interface{}{
			&role.EventID,
			&role.EventTitle,
			&role.EventOrganizerID,
			&role.EventOrganizerName,
			&role.SpaceID,
			&role.SpaceName,
			&role.VenueID,
			&role.VenueName,
			&role.OrganizerID,
			&role.OrganizerName,
			&role.UserRoleID,
			&role.RoleName,
		}
		scanTargets = append(scanTargets, permissionScanTargets(&role.Permissions)...)

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if existing, found := eventMap[role.EventID]; found {
			role.Permissions = mergePermissions(existing.Permissions, role.Permissions)
		}

		eventMap[role.EventID] = role
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return eventMap, nil
}

func TestQuery(gc *gin.Context) {
	dbPool := app.Singleton.MainDbPool
	userID, err := app.CurrentUserId(gc)
	if err != nil {
		gc.JSON(http.StatusUnauthorized, gin.H{"error": "user not logged in"})
		return
	}

	organizerRoles, err := FetchOrganizerRoles(gc, dbPool, userID)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	venueRoles, err := FetchVenueRoles(gc, dbPool, userID)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	spaceRoles, err := FetchSpaceRoles(gc, dbPool, userID)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	eventRoles, err := FetchEventRoles(gc, dbPool, userID)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	for _, role := range eventRoles {
		ensureOrganizerExists(organizerRoles, role.OrganizerID, role.OrganizerName)
		fmt.Println(role)
	}

	// Convert map to slice if preferred
	organizerList := make([]OrganizerRole, 0, len(organizerRoles))
	for _, role := range organizerRoles {
		organizerList = append(organizerList, role)
	}

	allRoles := AllRoles{
		OrganizerRoles: organizerRoles,
		VenueRoles:     venueRoles,
		SpaceRoles:     spaceRoles,
		EventRoles:     eventRoles,
	}

	err = Test2(gc, &allRoles)
	if err != nil {
		fmt.Println(err)
	}

	// gc.JSON(http.StatusOK, allRoles)
}

func Test2(gc *gin.Context, allRoles *AllRoles) error {
	ctx := gc.Request.Context()
	dbPool := app.Singleton.MainDbPool

	// Extract organizer IDs
	var organizerIDs []int
	for id := range allRoles.OrganizerRoles {
		organizerIDs = append(organizerIDs, id)
	}

	// Handle empty list (to avoid query error)
	if len(organizerIDs) == 0 {
		gc.JSON(http.StatusOK, gin.H{"stats": []any{}})
		return nil
	}

	sql := `SELECT
  o.id AS organizer_id,
  o.name AS organizer_name,
  COUNT(DISTINCT v.id) AS venue_count,
  COUNT(DISTINCT s.id) AS space_count,
  COUNT(DISTINCT ed.start) AS event_date_count
FROM uranus.organizer o
LEFT JOIN uranus.venue v ON v.organizer_id = o.id
LEFT JOIN uranus.space s ON s.venue_id = v.id
LEFT JOIN uranus.event e ON e.space_id = s.id
LEFT JOIN uranus.event_date ed ON ed.event_id = e.id AND ed.start >= '2020-08-06'
WHERE o.id = ANY($1)
GROUP BY o.id, o.name;`

	rows, err := dbPool.Query(ctx, sql, organizerIDs)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Collect results
	type Stats struct {
		OrganizerID    int    `json:"organizer_id"`
		OrganizerName  string `json:"organizer_name"`
		VenueCount     int    `json:"venue_count"`
		SpaceCount     int    `json:"space_count"`
		EventDateCount int    `json:"event_date_count"`
	}
	var stats []Stats

	for rows.Next() {
		var s Stats
		if err := rows.Scan(&s.OrganizerID, &s.OrganizerName, &s.VenueCount, &s.SpaceCount, &s.EventDateCount); err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}
		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration error: %w", err)
	}

	// Return JSON
	gc.JSON(http.StatusOK, gin.H{"stats": stats})
	return nil
}
