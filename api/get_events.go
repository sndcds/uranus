package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/sql_utils"
)

// eventType represents a type-genre mapping (example)
type eventType struct {
	TypeID    int    `json:"type_id"`
	TypeName  string `json:"type_name,omitempty"`
	GenreID   int    `json:"genre_id"`
	GenreName string `json:"genre_name,omitempty"`
}

// eventResponse is the JSON structure for each event
type eventResponse struct {
	EventDateID             int         `json:"event_date_id"`
	ID                      int         `json:"id"` // event_id
	Title                   string      `json:"title"`
	Subtitle                *string     `json:"subtitle"`
	Description             *string     `json:"description"`
	StartDate               string      `json:"start_date"`
	StartTime               string      `json:"start_time,omitempty"`
	EndDate                 *string     `json:"end_date,omitempty"`
	EndTime                 *string     `json:"end_time,omitempty"`
	EntryTime               *string     `json:"entry_time,omitempty"`
	Duration                *int        `json:"duration,omitempty"`
	AllDay                  *bool       `json:"all_day,omitempty"`
	Status                  *string     `json:"status,omitempty"`
	TicketLink              *string     `json:"ticket_link,omitempty"`
	SpaceID                 *int        `json:"space_id,omitempty"`
	SpaceName               *string     `json:"space_name,omitempty"`
	SpaceAccessibilityFlags *int64      `json:"space_accessibility_flags,omitempty"`
	VenueID                 *int        `json:"venue_id,omitempty"`
	VenueName               *string     `json:"venue_name,omitempty"`
	VenueCity               *string     `json:"venue_city,omitempty"`
	VenueStreet             *string     `json:"venue_street,omitempty"`
	VenueHouse              *string     `json:"venue_house_number,omitempty"`
	VenuePostal             *string     `json:"venue_postal_code,omitempty"`
	VenueState              *string     `json:"venue_state_code,omitempty"`
	VenueCountry            *string     `json:"venue_country_code,omitempty"`
	VenueLat                *float64    `json:"venue_lat,omitempty"`
	VenueLon                *float64    `json:"venue_lon,omitempty"`
	ImageId                 *int        `json:"image_id,omitempty"`
	ImagePath               *string     `json:"image_path,omitempty"`
	OrganizationID          int         `json:"organization_id"`
	OrganizationName        string      `json:"organization_name"`
	EventTypes              []eventType `json:"event_types,omitempty"`
	Languages               []string    `json:"languages"`
	Tags                    []string    `json:"tags"`
	MinAge                  *int        `json:"min_age"`
	MaxAge                  *int        `json:"max_age"`
	VisitorInfoFlags        *int64      `json:"visitor_info_flags,omitempty"`
	// Add other fields as needed
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
	nextArgIndex int,
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
	nextArgIndex = 1
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
	accessibilityInfosStr, _ := GetContextParam(gc, "accessibility")
	visitorInfosStr, _ := GetContextParam(gc, "visitor_infos")
	ageStr, _ := GetContextParam(gc, "age")
	lonStr, _ := GetContextParam(gc, "lon")
	latStr, _ := GetContextParam(gc, "lat")
	radiusStr, _ := GetContextParam(gc, "radius")
	offsetStr, _ := GetContextParam(gc, "offset")
	limitStr, _ := GetContextParam(gc, "limit")

	// --- date conditions ---
	dateConditionCount := 0
	if app.IsValidDateStr(startStr) {
		dateConditions += "edp.event_start_at >= $" + strconv.Itoa(nextArgIndex)
		args = append(args, startStr)
		nextArgIndex++
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
		dateConditions += "(edp.event_end_at <= $" + strconv.Itoa(nextArgIndex) + " OR edp.event_start_at <= $" + strconv.Itoa(nextArgIndex) + ")"
		args = append(args, endStr)
		nextArgIndex++
	} else if endStr != "" {
		return "", "", "", nil, 0, fmt.Errorf("end %s has invalid format", endStr)
	}

	if lastEventStartAt != "" {
		if dateConditionCount > 0 {
			dateConditions += " AND "
		}
		dateConditions += "(edp.event_start_at, edp.event_date_id) > ($" + strconv.Itoa(nextArgIndex) + "::timestamptz, $" + strconv.Itoa(nextArgIndex+1) + "::int)"
		args = append(args, lastEventStartAt, lastEventDateId)
		nextArgIndex += 2
	}

	// --- other conditions ---
	var errBuild error
	nextArgIndex, errBuild = sql_utils.BuildTimeCondition(timeStr, "edp.start_time", "time", nextArgIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}
	nextArgIndex, errBuild = sql_utils.BuildSanitizedSearchCondition(searchStr, "ep.search_text", "search", nextArgIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}
	nextArgIndex, errBuild = sql_utils.BuildSanitizedIlikeCondition(titleStr, "ep.title", "title", nextArgIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	if countryCodesStr != "" {
		format := "COALESCE(edp.venue_country_code, ep.venue_country_code) = ANY(%s)"
		nextArgIndex, errBuild = sql_utils.BuildInConditionForStringSlice(countryCodesStr, format, "countries", nextArgIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}
	if postalCodeStr != "" {
		nextArgIndex, errBuild = sql_utils.BuildLikeConditions(postalCodeStr, "COALESCE(edp.venue_postal_code, ep.venue_postal_code)", nextArgIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}
	nextArgIndex, errBuild = sql_utils.BuildSanitizedIlikeCondition(cityStr, "COALESCE(edp.venue_city, ep.venue_city)", "city", nextArgIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}
	if eventIdsStr != "" {
		nextArgIndex, errBuild = sql_utils.BuildColumnInIntCondition(eventIdsStr, "edp.event_id", nextArgIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}
	if organizationIdsStr != "" {
		nextArgIndex, errBuild = sql_utils.BuildColumnInIntCondition(organizationIdsStr, "ep.organization_id", nextArgIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}
	if venueIdsStr != "" {
		nextArgIndex, errBuild = sql_utils.BuildColumnInIntCondition(venueIdsStr, "COALESCE(edp.venue_id, ep.venue_id)", nextArgIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}
	if spaceIdsStr != "" {
		nextArgIndex, errBuild = sql_utils.BuildColumnInIntCondition(spaceIdsStr, "COALESCE(edp.space_id, ep.space_id)", nextArgIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}
	if spaceTypesStr != "" {
		nextArgIndex, errBuild = sql_utils.BuildColumnInIntCondition(spaceTypesStr, "COALESCE(edp.space_type_id, ep.space_type_id)", nextArgIndex, &conditions, &args)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}
	nextArgIndex, errBuild = sql_utils.BuildGeoRadiusCondition(lonStr, latStr, radiusStr, "COALESCE(edp.venue_geo_pos, ep.venue_geo_pos)", nextArgIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}
	nextArgIndex, errBuild = sql_utils.BuildContainedInColumnIntRangeCondition(ageStr, "ep.min_age", "ep.max_age", nextArgIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}
	nextArgIndex, errBuild = sql_utils.BuildBitmaskCondition(accessibilityInfosStr, "edp.space_accessibility_flags", "accessibility", nextArgIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}
	nextArgIndex, errBuild = sql_utils.BuildBitmaskCondition(visitorInfosStr, "edp.visitor_info_flags", "visitor_infos", nextArgIndex, &conditions, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	if eventTypesStr != "" {
		nextArgIndex, errBuild = sql_utils.BuildJsonbArrayIntCondition(
			eventTypesStr,
			"types",
			0, // index 0 = event_type_id
			nextArgIndex,
			&conditions,
			&args,
		)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	if tagsStr != "" {
		nextArgIndex, errBuild = sql_utils.BuildArrayContainsCondition(
			tagsStr,
			"tags",
			nextArgIndex,
			&conditions,
			&args,
		)
		if errBuild != nil {
			return "", "", "", nil, 0, errBuild
		}
	}

	// join all conditions
	if len(conditions) > 0 {
		conditionsStr = " AND " + strings.Join(conditions, " AND ")
	}

	// build limit/offset clause
	limitClause, nextArgIndex, errBuild = sql_utils.BuildLimitOffsetClause(limitStr, offsetStr, nextArgIndex, &args)
	if errBuild != nil {
		return "", "", "", nil, 0, errBuild
	}

	return dateConditions, conditionsStr, limitClause, args, nextArgIndex, nil
}

func (h *ApiHandler) GetEvents(gc *gin.Context) {
	ctx := gc.Request.Context()

	dateConditions, conditionsStr, limitClause, args, _, err := h.buildEventFilters(gc)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := app.Singleton.SqlGetEventsProjected
	query = strings.Replace(query, "{{date_conditions}}", dateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", conditionsStr, 1)
	query = strings.Replace(query, "{{limit}}", limitClause, 1)

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
			&e.EventDateID,
			&e.ID,
			&e.OrganizationID,
			&e.VenueID,
			&e.SpaceID,
			&e.StartDate,
			&e.StartTime,
			&e.EndDate,
			&e.EndTime,
			&e.EntryTime,
			&e.Duration,
			&e.AllDay,
			&e.Status,
			&e.TicketLink,
			&e.Title,
			&e.Subtitle,
			&e.Description,
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
					TypeID:  pair[0],
					GenreID: pair[1],
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
	lastEventDateID := lastEvent.EventDateID

	gc.JSON(http.StatusOK, gin.H{
		"events":              events,
		"last_event_start_at": lastEventStartAt,
		"last_event_date_id":  lastEventDateID,
	})
}

func (h *ApiHandler) GetEventTypeDateSummary(gc *gin.Context) {
	dateConditions, conditionsStr, _, args, _, err := h.buildEventFilters(gc)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := fmt.Sprintf(`
WITH upcoming_dates AS (
    SELECT *
    FROM %s.event_date_projection edp
    WHERE %s
)
SELECT
    (elem->>0)::int AS type_id,
    COUNT(edp.event_date_id) AS date_count
FROM upcoming_dates edp
JOIN %s.event_projection ep ON ep.event_id = edp.event_id
CROSS JOIN LATERAL jsonb_array_elements(ep.types) AS elem
WHERE ep.release_status_id >= 3
%s
GROUP BY type_id
ORDER BY date_count DESC;
`, h.DbSchema, dateConditions, h.DbSchema, conditionsStr)

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

func validateAllowedQueryParams(c *gin.Context, allowed map[string]struct{}) error {
	for key := range c.Request.URL.Query() {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("unsupported query parameter: %s", key)
		}
	}
	return nil
}
