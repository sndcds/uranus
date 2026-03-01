package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/sql_utils"
)

// eventType represents a type-genre mapping (example)
type eventType struct {
	TypeId  int `json:"type_id"`
	GenreId int `json:"genre_id"`
}

// eventResponse is the JSON structure for each event
type eventResponse struct {
	EventDateId             int         `json:"event_date_id"`
	Id                      int         `json:"id"` // event_id
	Title                   string      `json:"title"`
	Subtitle                *string     `json:"subtitle"`
	StartDate               string      `json:"start_date"`
	StartTime               string      `json:"start_time,omitempty"`
	EndDate                 *string     `json:"end_date,omitempty"`
	EndTime                 *string     `json:"end_time,omitempty"`
	EntryTime               *string     `json:"entry_time,omitempty"`
	Duration                *int        `json:"duration,omitempty"`
	AllDay                  *bool       `json:"all_day,omitempty"`
	TicketLink              *string     `json:"ticket_link,omitempty"`
	SpaceId                 *int        `json:"space_id,omitempty"`
	SpaceName               *string     `json:"space_name,omitempty"`
	SpaceAccessibilityFlags *string     `json:"space_accessibility_flags,omitempty"`
	VenueId                 *int        `json:"venue_id,omitempty"`
	VenueName               *string     `json:"venue_name,omitempty"`
	VenueCity               *string     `json:"venue_city,omitempty"`
	VenueStreet             *string     `json:"venue_street,omitempty"`
	VenueHouse              *string     `json:"venue_house_number,omitempty"`
	VenuePostal             *string     `json:"venue_postal_code,omitempty"`
	VenueState              *string     `json:"venue_state,omitempty"`
	VenueCountry            *string     `json:"venue_country,omitempty"`
	VenueLat                *float64    `json:"venue_lat,omitempty"`
	VenueLon                *float64    `json:"venue_lon,omitempty"`
	ImageId                 *int        `json:"image_id,omitempty"`
	ImagePath               *string     `json:"image_path,omitempty"`
	OrganizationId          int         `json:"organization_id"`
	OrganizationName        string      `json:"organization_name"`
	EventTypes              []eventType `json:"event_types,omitempty"`
	Languages               []string    `json:"languages"`
	Tags                    []string    `json:"tags"`
	MinAge                  *int        `json:"min_age"`
	MaxAge                  *int        `json:"max_age"`
	VisitorInfoFlags        *string     `json:"visitor_info_flags,omitempty"`
	ReleaseStatus           *string     `json:"release_status,omitempty"`
}

// buildEventFilters parses all query parameters from the context
// and returns:
// - dateConditions: the date-specific condition string
// - conditionsStr: all other conditions concatenated
// - limitClause: SQL LIMIT/OFFSET clause
// - args: list of query arguments
// - nextArgIndex: next placeholder index
func (h *ApiHandler) buildEventFilters(gc *gin.Context) (
	dateConditions string,
	conditionsStr string,
	limitClause string,
	args []interface{},
	argIndex int,
	err error,
) {

	allowed := map[string]struct{}{
		"past": {}, "start": {}, "end": {}, "time": {}, "search": {},
		"events": {}, "venues": {}, "spaces": {}, "space_types": {},
		"organizations": {}, "countries": {}, "postal_code": {},
		"title": {}, "city": {}, "event_types": {}, "tags": {},
		"accessibility": {}, "visitor_infos": {}, "age": {},
		"lon": {}, "lat": {}, "radius": {}, "offset": {}, "limit": {},
		"last_event_start_at": {}, "last_event_date_id": {},
		"language": {},
	}

	validationErr := validateAllowedQueryParams(gc, allowed)
	if validationErr != nil {
		return "", "", "", nil, 0, fmt.Errorf(validationErr.Error())
	}
	args = []interface{}{}
	argIndex = 1
	var conditions []string

	_, hasPast := GetContextParam(gc, "past")
	// languagesStr, _ := GetContextParam(gc, "language") // TODO: Implement language support!
	startStr, _ := GetContextParam(gc, "start")
	endStr, _ := GetContextParam(gc, "end")
	lastEventStartAt, _ := GetContextParam(gc, "last_event_start_at")
	lastEventDateId := GetContextParamIntDefault(gc, "last_event_date_id", 0)
	timeStr, _ := GetContextParam(gc, "time")
	searchStr, _ := GetContextParam(gc, "search")
	eventIdsStr, _ := GetContextParam(gc, "events")
	venueIdsStr, _ := GetContextParam(gc, "venues")
	spaceIdsStr, _ := GetContextParam(gc, "spaces")
	spaceTypesStr, _ := GetContextParam(gc, "space_types")
	organizationIdsStr, _ := GetContextParam(gc, "organizations")
	countryCodesStr, _ := GetContextParam(gc, "countries")
	postalCodeStr, _ := GetContextParam(gc, "postal_code")
	titleStr, _ := GetContextParam(gc, "title")
	cityStr, _ := GetContextParam(gc, "city")
	eventTypesStr, _ := GetContextParam(gc, "event_types")
	tagsStr, _ := GetContextParam(gc, "tags")
	accessibilityFlagsStr, _ := GetContextParam(gc, "accessibility")
	visitorInfosStr, _ := GetContextParam(gc, "visitor_infos")
	ageStr, _ := GetContextParam(gc, "age")
	lonStr, _ := GetContextParam(gc, "lon")
	latStr, _ := GetContextParam(gc, "lat")
	radiusStr, _ := GetContextParam(gc, "radius")
	offsetStr, _ := GetContextParam(gc, "offset")
	limitStr, _ := GetContextParam(gc, "limit")

	// Date conditions
	dateConditionCount := 0
	if app.IsValidDateStr(startStr) {
		dateConditions += "edp.event_start_at >= $" + strconv.Itoa(argIndex)
		args = append(args, startStr)
		argIndex++
		dateConditionCount++
	} else if startStr != "" {
		return "", "", "", nil, 0, fmt.Errorf("start %s has invalid format", startStr)
	} else if !hasPast {
		dateConditions += "edp.event_start_at >= CURRENT_DATE"
		dateConditionCount++
	}

	if app.IsValidDateStr(endStr) {
		if dateConditionCount > 0 {
			dateConditions += " AND "
		}
		dateConditions += "(edp.event_end_at <= $" + strconv.Itoa(argIndex) + " OR edp.event_start_at <= $" + strconv.Itoa(argIndex) + ")"
		args = append(args, endStr)
		argIndex++
	} else if endStr != "" {
		return "", "", "", nil, 0, fmt.Errorf("end %s has invalid format", endStr)
	}

	if lastEventStartAt != "" {
		if dateConditionCount > 0 {
			dateConditions += " AND "
		}
		dateConditions += "(edp.event_start_at, edp.event_date_id) > ($" + strconv.Itoa(argIndex) + "::timestamptz, $" + strconv.Itoa(argIndex+1) + "::int)"
		args = append(args, lastEventStartAt, lastEventDateId)
		argIndex += 2
	}

	// Other conditions
	var errBuild error
	argIndex, errBuild = sql_utils.BuildTimeCondition(
		timeStr, "edp.start_time", "time", argIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	argIndex, errBuild = sql_utils.BuildSanitizedSearchCondition(
		searchStr, "ep.search_text", "search", argIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	argIndex, errBuild = sql_utils.BuildSanitizedIlikeCondition(
		titleStr, "ep.title", "title", argIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	if countryCodesStr != "" {
		argIndex, errBuild = sql_utils.BuildInConditionForStringSlice(
			countryCodesStr,
			"COALESCE(edp.venue_country, ep.venue_country) = ANY($%d::text[])", // "column_name && $%d::text[]",
			argIndex,
			&conditions,
			&args,
		)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	if postalCodeStr != "" {
		argIndex, errBuild = sql_utils.BuildLikeConditions(
			postalCodeStr,
			"COALESCE(edp.venue_postal_code, ep.venue_postal_code)",
			argIndex,
			&conditions,
			&args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	argIndex, errBuild = sql_utils.BuildSanitizedIlikeCondition(
		cityStr, "COALESCE(edp.venue_city, ep.venue_city)",
		"city", argIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}
	if eventIdsStr != "" {
		argIndex, errBuild = sql_utils.BuildColumnInIntCondition(
			eventIdsStr, "edp.event_id", argIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	if organizationIdsStr != "" {
		argIndex, errBuild = sql_utils.BuildColumnInIntCondition(
			organizationIdsStr, "ep.organization_id", argIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	if venueIdsStr != "" {
		argIndex, errBuild = sql_utils.BuildColumnInIntCondition(
			venueIdsStr, "COALESCE(edp.venue_id, ep.venue_id)", argIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	if spaceIdsStr != "" {
		argIndex, errBuild = sql_utils.BuildColumnInIntCondition(
			spaceIdsStr, "COALESCE(edp.space_id, ep.space_id)", argIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	if spaceTypesStr != "" {
		argIndex, errBuild = sql_utils.BuildInConditionForStringSlice(
			spaceTypesStr,
			"COALESCE(edp.space_type, ep.space_type) = ANY($%d::text[])",
			argIndex, &conditions, &args,
		)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	argIndex, errBuild = sql_utils.BuildGeoRadiusCondition(
		lonStr, latStr, radiusStr,
		"COALESCE(edp.venue_geo_pos, ep.venue_geo_pos)",
		argIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	argIndex, errBuild = sql_utils.BuildContainedInColumnIntRangeCondition(
		ageStr, "ep.min_age", "ep.max_age", argIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	argIndex, errBuild = sql_utils.BuildBitmaskCondition(
		accessibilityFlagsStr, "edp.space_accessibility_flags", "accessibility", argIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	argIndex, errBuild = sql_utils.BuildBitmaskCondition(
		visitorInfosStr, "ep.visitor_info_flags", "visitor_infos", argIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	if eventTypesStr != "" {
		argIndex, errBuild = sql_utils.BuildJSONArrayIntCondition(
			eventTypesStr,
			"types",
			0, // index 0 = type_id
			argIndex,
			&conditions,
			&args,
		)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	if tagsStr != "" {
		argIndex, errBuild = sql_utils.BuildInConditionForStringSlice(
			tagsStr,
			"tags && $%d::text[]", // Array overlap operator
			argIndex,
			&conditions,
			&args,
		)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	// Join all conditions
	if len(conditions) > 0 {
		conditionsStr = " AND " + strings.Join(conditions, " AND ")
	}

	// Build limit/offset clause
	limitClause, argIndex, errBuild = sql_utils.BuildLimitOffsetClause(limitStr, offsetStr, argIndex, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	return dateConditions, conditionsStr, limitClause, args, argIndex, nil
}

func (h *ApiHandler) GetEvents(gc *gin.Context) {
	ctx := gc.Request.Context()

	dateConditions, conditionsStr, limitClause, args, _, err := h.buildEventFilters(gc)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := app.UranusInstance.SqlGetEventsProjected
	query = strings.Replace(query, "{{date_conditions}}", dateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", conditionsStr, 1)
	query = strings.Replace(query, "{{limit}}", limitClause, 1)

	fmt.Println(query)
	fmt.Printf("ARGS (%d):\n", len(args))
	for i, arg := range args {
		fmt.Printf("  $%d = %#v (type %T)\n", i+1, arg, arg)
	}

	rows, err := h.DbPool.Query(ctx, query, args...)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("query failed: %v", err)})
		return
	}
	defer rows.Close()

	var events []eventResponse

	for rows.Next() {
		var e eventResponse
		var typesJSON []byte
		err := rows.Scan(
			&e.EventDateId,
			&e.Id,
			&e.OrganizationId,
			&e.VenueId,
			&e.SpaceId,
			&e.StartDate,
			&e.StartTime,
			&e.EndDate,
			&e.EndTime,
			&e.EntryTime,
			&e.Duration,
			&e.AllDay,
			&e.ReleaseStatus,
			&e.TicketLink,
			&e.Title,
			&e.Subtitle,
			&typesJSON,
			&e.Languages,
			&e.Tags,
			&e.OrganizationName,
			&e.ImageId,
			&e.VenueName,
			&e.VenueCity,
			&e.VenueStreet,
			&e.VenueHouse,
			&e.VenuePostal,
			&e.VenueState,
			&e.VenueCountry,
			&e.VenueLat,
			&e.VenueLon,
			&e.SpaceName,
			&e.SpaceAccessibilityFlags,
			&e.MinAge,
			&e.MaxAge,
			&e.VisitorInfoFlags,
		)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("scan failed: %v", err)})
			return
		}

		// Convert types JSON
		var rawTypes [][]int
		if len(typesJSON) > 0 {
			if err := json.Unmarshal(typesJSON, &rawTypes); err != nil {
				gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("json unmarshal failed (types): %v", err)})
				return
			}
			e.EventTypes = make([]eventType, len(rawTypes))
			for i, pair := range rawTypes {
				e.EventTypes[i] = eventType{
					TypeId:  pair[0],
					GenreId: pair[1],
				}
			}
		} else {
			e.EventTypes = []eventType{}
		}

		if e.ImageId != nil {
			path := fmt.Sprintf("%s/api/image/%d", h.Config.BaseApiUrl, *e.ImageId)
			e.ImagePath = &path
		}

		events = append(events, e)
	}

	if len(events) == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"events": events})
		return
	}

	lastEvent := events[len(events)-1]
	lastEventStartAt := lastEvent.StartDate + "T" + lastEvent.StartTime
	lastEventDateId := lastEvent.EventDateId

	gc.JSON(http.StatusOK, gin.H{
		"events":              events,
		"last_event_start_at": lastEventStartAt,
		"last_event_date_id":  lastEventDateId,
	})
}

func (h *ApiHandler) GetEventTypeSummary(gc *gin.Context) {
	// Build filters from query params (same as GetEvents)
	dateConditions, conditionsStr, _, args, _, err := h.buildEventFilters(gc)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Construct final SQL
	query := fmt.Sprintf(`
SELECT
    (elem->>0)::int AS type_id,
    COUNT(edp.event_date_id) AS date_count
FROM %s.event_date_projection edp
JOIN %s.event_projection ep ON ep.event_id = edp.event_id
CROSS JOIN LATERAL jsonb_array_elements(ep.types) AS elem
WHERE ep.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled')    
AND {{date_conditions}}
{{conditions}}
GROUP BY type_id
ORDER BY date_count DESC`,
		h.DbSchema, h.DbSchema)

	query = strings.Replace(query, "{{date_conditions}}", dateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", conditionsStr, 1)
	// query = strings.Replace(query, "{{limit}}", limitClause, 1)

	rows, err := h.DbPool.Query(gc.Request.Context(), query, args...)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type summaryEntry struct {
		TypeID    int `json:"type_id"`
		DateCount int `json:"date_count"`
	}

	var summary []summaryEntry
	for rows.Next() {
		var s summaryEntry
		if err := rows.Scan(&s.TypeID, &s.DateCount); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		summary = append(summary, s)
	}

	gc.JSON(http.StatusOK, gin.H{"summary": summary})
}

func (h *ApiHandler) GetEventVenueSummary(gc *gin.Context) {
	dateConditions, conditionsStr, _, args, _, err := h.buildEventFilters(gc)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	conds := []string{"ep.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled'"}

	if dateConditions != "" {
		conds = append(conds, dateConditions)
	}

	if conditionsStr != "" {
		conds = append(conds, conditionsStr)
	}

	query := fmt.Sprintf(`
SELECT jsonb_agg(
    jsonb_build_object(
        'venue_id', venue_id,
        'venue_name', venue_name,
        'date_count', date_count
    )
    ORDER BY venue_name ASC
) AS venues
FROM (
    SELECT
        COALESCE(edp.venue_id, ep.venue_id) AS venue_id,
        COALESCE(edp.venue_name, ep.venue_name) AS venue_name,
        COUNT(edp.event_date_id) AS date_count
    FROM %s.event_date_projection edp
    JOIN %s.event_projection ep
      ON ep.event_id = edp.event_id
    WHERE %s
    GROUP BY COALESCE(edp.venue_id, ep.venue_id),
             COALESCE(edp.venue_name, ep.venue_name)
) AS venue_counts`,
		h.DbSchema, h.DbSchema, strings.Join(conds, " AND "))

	var jsonResult []byte
	err = h.DbPool.QueryRow(gc.Request.Context(), query, args...).Scan(&jsonResult)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// jsonResult is already JSON; unmarshal to interface{} to let gin render it
	var venues interface{}
	if err := json.Unmarshal(jsonResult, &venues); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"venue-summary": venues})
}

func (h *ApiHandler) GetEventsGeoJSON(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "get-events-geojson")

	// Build SQL
	dateConditions, conditionsStr, limitClause, args, _, err := h.buildEventFilters(gc)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, "")
		return
	}

	query := app.UranusInstance.SqlGetEventsGeoJSON
	query = strings.Replace(query, "{{date_conditions}}", dateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", conditionsStr, 1)
	query = strings.Replace(query, "{{limit}}", limitClause, 1)

	debugf(query)
	debugf("ARGS (%d):\n", len(args))
	for i, arg := range args {
		fmt.Printf("  $%d = %#v (type %T)\n", i+1, arg, arg)
	}

	rows, err := h.DbPool.Query(ctx, query, args...)
	if err != nil {
		debugf("internal server error: %v", err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	// Event scan type
	type EventResponse struct {
		EventDateId  int      `json:"event_date_id"`
		EventId      int      `json:"event_id"`
		VenueId      *int     `json:"venue_id"`
		VenueName    *string  `json:"venue_name"`
		VenueCity    *string  `json:"venue_city"`
		VenueCountry *string  `json:"venue_country"`
		VenueLat     *float64 `json:"venue_lat"`
		VenueLon     *float64 `json:"venue_lon"`
		Title        string   `json:"title"`
		StartDate    string   `json:"start_date"`
		StartTime    *string  `json:"start_time"`
	}

	type VenueEvents struct {
		Name       string          `json:"name"`
		Lon        *float64        `json:"lon"`
		Lat        *float64        `json:"lat"`
		Events     []EventResponse `json:"events"`
		EventCount int             `json:"event_count"`
	}

	venues := make(map[int]*VenueEvents)

	var events []EventResponse

	for rows.Next() {
		var e EventResponse
		if err := rows.Scan(
			&e.EventDateId,
			&e.EventId,
			&e.VenueId,
			&e.VenueName,
			&e.VenueCity,
			&e.VenueCountry,
			&e.VenueLon,
			&e.VenueLat,
			&e.Title,
			&e.StartDate,
			&e.StartTime,
		); err != nil {
			debugf("internal server error: %v", err.Error())
			apiRequest.InternalServerError()
			return
		}

		events = append(events, e)

		// Skip events without a venue
		if e.VenueId == nil {
			continue
		}

		if e.VenueId != nil {
			vId := *e.VenueId
			if _, exists := venues[vId]; !exists {
				venues[vId] = &VenueEvents{
					Name:   derefString(e.VenueName, ""),
					Lon:    e.VenueLon,
					Lat:    e.VenueLat,
					Events: []EventResponse{},
					// don't set EventCount yet
				}
			}

			venues[vId].Events = append(venues[vId].Events, e)
			venues[vId].EventCount = len(venues[vId].Events)
		}
	}

	if len(events) == 0 {
		apiRequest.NoContent("no events found")
		return
	}

	apiRequest.SetMeta("event_count", len(events))

	lastEvent := events[len(events)-1]
	lastEventStartAt := lastEvent.StartDate
	if lastEvent.StartTime != nil {
		lastEventStartAt += "T" + *lastEvent.StartTime
	}
	lastEventDateId := lastEvent.EventDateId

	apiRequest.Success(
		http.StatusOK,
		gin.H{
			"venues":              venues,
			"last_event_start_at": lastEventStartAt,
			"last_event_date_id":  lastEventDateId,
		},
		"",
	)
}

func validateAllowedQueryParams(c *gin.Context, allowed map[string]struct{}) error {
	for key := range c.Request.URL.Query() {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("unsupported query parameter: %s", key)
		}
	}
	return nil
}

// Helper for nil strings
func derefString(s *string, fallback string) string {
	if s != nil && *s != "" {
		return *s
	}
	return fallback
}
