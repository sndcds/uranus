package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	Uuid                    string      `json:"uuid"`
	DateUuid                string      `json:"date_uuid"`
	DateSlug                string      `json:"date_slug"`
	Title                   string      `json:"title"`
	Subtitle                *string     `json:"subtitle"`
	Summary                 *string     `json:"summary"`
	StartDate               string      `json:"start_date"`
	StartTime               string      `json:"start_time,omitempty"`
	EndDate                 *string     `json:"end_date,omitempty"`
	EndTime                 *string     `json:"end_time,omitempty"`
	EntryTime               *string     `json:"entry_time,omitempty"`
	Duration                *int        `json:"duration,omitempty"`
	AllDay                  *bool       `json:"all_day,omitempty"`
	TicketLink              *string     `json:"ticket_link,omitempty"`
	SpaceUuid               *string     `json:"space_uuid,omitempty"`
	SpaceName               *string     `json:"space_name,omitempty"`
	SpaceAccessibilityFlags *string     `json:"space_accessibility_flags,omitempty"`
	VenueUuid               *string     `json:"venue_uuid,omitempty"`
	VenueName               *string     `json:"venue_name,omitempty"`
	VenueCity               *string     `json:"venue_city,omitempty"`
	VenueStreet             *string     `json:"venue_street,omitempty"`
	VenueHouse              *string     `json:"venue_house_number,omitempty"`
	VenuePostal             *string     `json:"venue_postal_code,omitempty"`
	VenueState              *string     `json:"venue_state,omitempty"`
	VenueCountry            *string     `json:"venue_country,omitempty"`
	VenueLat                *float64    `json:"venue_lat,omitempty"`
	VenueLon                *float64    `json:"venue_lon,omitempty"`
	ImageUuid               *string     `json:"image_uuid,omitempty"`
	ImagePath               *string     `json:"image_path,omitempty"`
	OrgUuid                 string      `json:"org_uuid"`
	OrgName                 string      `json:"org_name"`
	Categories              []int       `json:"categories,omitempty"`
	EventTypes              []eventType `json:"event_types,omitempty"`
	Languages               []string    `json:"languages"`
	Tags                    []string    `json:"tags"`
	MinAge                  *int        `json:"min_age"`
	MaxAge                  *int        `json:"max_age"`
	PriceType               *string     `json:"price_type,omitempty"`
	VisitorInfoFlags        *string     `json:"visitor_info_flags,omitempty"`
	ReleaseStatus           *string     `json:"release_status,omitempty"`
}

type eventsResponse struct {
	Events            []eventResponse `json:"events"`
	LastEventDateUuid *string         `json:"last_event_date_uuid"`
	LastEventStartAt  *string         `json:"last_event_start_at"`
}

type eventFilters struct {
	WeekStart        string
	DateConditions   string
	ConditionsStr    string
	LimitClause      string
	PortalJoin       string
	PortalConditions string
	Args             []interface{}
	ArgIndex         int
}

// buildEventFilters parses all query parameters from the context
// and returns:
// - dateConditions: the date-specific condition string
// - conditionsStr: all other conditions concatenated
// - limitClause: SQL LIMIT/OFFSET clause
// - args: list of query arguments
// - nextArgIndex: next placeholder index
func (h *ApiHandler) buildEventFilters(gc *gin.Context, useTypeFilter bool) (eventFilters, error) {
	allowed := map[string]struct{}{
		"categories":           {},
		"start":                {},
		"end":                  {},
		"time":                 {},
		"search":               {},
		"events":               {},
		"venue_uuids":          {},
		"venue":                {},
		"space_uuids":          {},
		"space_types":          {},
		"org_uuids":            {},
		"countries":            {},
		"postal_code":          {},
		"title":                {},
		"city":                 {},
		"event_types":          {},
		"genres":               {},
		"tags":                 {},
		"accessibility":        {},
		"visitor_infos":        {},
		"age":                  {},
		"price":                {},
		"lon":                  {},
		"lat":                  {},
		"radius":               {},
		"offset":               {},
		"limit":                {},
		"last_event_start_at":  {},
		"last_event_date_uuid": {},
		"lang":                 {},
		"portal":               {},
		"week_start":           {},
		"geolist_region":       {},
	}

	filters := eventFilters{}

	validationErr := validateAllowedQueryParams(gc, allowed)
	if validationErr != nil {
		return filters, errors.New(validationErr.Error())
	}
	filters.Args = []interface{}{}
	filters.ArgIndex = 1
	var conditions []string

	var eventTypesStr string
	var genresStr string

	// langStr, _ := GetContextParam(gc, "language") // TODO: Implement language support!
	categoriesStr, hasCategories := GetContextParam(gc, "categories")
	startStr, _ := GetContextParam(gc, "start")
	endStr, _ := GetContextParam(gc, "end")
	lastEventStartAt, _ := GetContextParam(gc, "last_event_start_at")
	lastEventDateUuid, _ := GetContextParam(gc, "last_event_date_uuid")
	timeStr, _ := GetContextParam(gc, "time")
	searchStr, _ := GetContextParam(gc, "search")
	eventUuidsStr, _ := GetContextParam(gc, "events")
	venueStr, _ := GetContextParam(gc, "venue")
	venueUuidsStr, _ := GetContextParam(gc, "venue_uuids")
	spaceUuidsStr, _ := GetContextParam(gc, "space_uuids")
	spaceTypesStr, _ := GetContextParam(gc, "space_types")
	orgUuidsStr, _ := GetContextParam(gc, "org_uuids")
	countryCodesStr, _ := GetContextParam(gc, "countries")
	postalCodeStr, _ := GetContextParam(gc, "postal_code")
	titleStr, _ := GetContextParam(gc, "title")
	cityStr, _ := GetContextParam(gc, "city")
	if useTypeFilter {
		eventTypesStr, _ = GetContextParam(gc, "event_types")
		genresStr, _ = GetContextParam(gc, "genres")
		if eventTypesStr != "" && genresStr != "" {
			return filters, fmt.Errorf("only one of 'event_types' or 'genres' may be specified")
		}
	}
	tagsStr, _ := GetContextParam(gc, "tags")
	accessibilityFlagsStr, _ := GetContextParam(gc, "accessibility")
	visitorInfosStr, _ := GetContextParam(gc, "visitor_infos")
	ageStr, _ := GetContextParam(gc, "age")
	priceStr, _ := GetContextParam(gc, "price")
	lonStr, _ := GetContextParam(gc, "lon")
	latStr, _ := GetContextParam(gc, "lat")
	radiusStr, _ := GetContextParam(gc, "radius")
	offsetStr, _ := GetContextParam(gc, "offset")
	limitStr, _ := GetContextParam(gc, "limit")
	weekStartStr, _ := GetContextParam(gc, "week_start")
	geolistRegionStr, _ := GetContextParam(gc, "geolist_region")

	var errBuild error

	if hasCategories {
		filters.ArgIndex, errBuild = sql_utils.BuildColumnArrayOverlapCondition(
			categoriesStr, "ep.categories", filters.ArgIndex, &conditions, &filters.Args)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	// Date conditions
	dateConditionCount := 0
	if app.IsValidDateStr(startStr) {
		filters.DateConditions += "COALESCE(edp.event_end_at, edp.event_start_at) >= $" + strconv.Itoa(filters.ArgIndex)
		filters.Args = append(filters.Args, startStr)
		filters.ArgIndex++
		dateConditionCount++
	} else if startStr != "" {
		return filters, fmt.Errorf("'start' has invalid format: %s", startStr)
	} else {
		filters.DateConditions += "COALESCE(edp.event_end_at, edp.event_start_at) >= CURRENT_DATE"
		dateConditionCount++
	}

	if app.IsValidDateStr(endStr) {
		endDate, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return filters, fmt.Errorf("end %s has invalid format", endStr)
		}
		endDate = endDate.AddDate(0, 0, 1)
		if dateConditionCount > 0 {
			filters.DateConditions += " AND "
		}
		filters.DateConditions += "(" +
			"edp.event_end_at < $" + strconv.Itoa(filters.ArgIndex) +
			" OR edp.event_start_at < $" + strconv.Itoa(filters.ArgIndex) +
			")"
		filters.Args = append(filters.Args, endDate)
		filters.ArgIndex++

	} else if endStr != "" {
		return filters, fmt.Errorf("end %s has invalid format", endStr)
	}

	if lastEventStartAt != "" {
		if dateConditionCount > 0 {
			filters.DateConditions += " AND "
		}
		filters.DateConditions += "(edp.event_start_at, edp.event_date_uuid) > ($" + strconv.Itoa(filters.ArgIndex) + "::timestamptz, $" + strconv.Itoa(filters.ArgIndex+1) + "::uuid)"
		filters.Args = append(filters.Args, lastEventStartAt, lastEventDateUuid)
		filters.ArgIndex += 2
	}

	// debugf("dateConditions: %s", filters.DateConditions)

	// Other conditions
	filters.ArgIndex, errBuild = sql_utils.BuildTimeCondition(
		timeStr, "edp.start_time", "time", filters.ArgIndex, &conditions, &filters.Args)
	if errBuild != nil {
		return filters, errBuild
	}

	filters.ArgIndex, errBuild = sql_utils.BuildSanitizedSearchCondition(
		searchStr, "ep.search_text", "search", filters.ArgIndex, &conditions, &filters.Args)
	if errBuild != nil {
		return filters, errBuild
	}

	filters.ArgIndex, errBuild = sql_utils.BuildSanitizedIlikeCondition(
		titleStr, "ep.title", "title", filters.ArgIndex, &conditions, &filters.Args)
	if errBuild != nil {
		return filters, errBuild
	}

	if countryCodesStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildInConditionForStringSlice(
			countryCodesStr,
			"COALESCE(edp.venue_country, ep.venue_country) = ANY($%d::text[])", // "column_name && $%d::text[]",
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if postalCodeStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildLikeConditions(
			postalCodeStr,
			"COALESCE(edp.venue_postal_code, ep.venue_postal_code)",
			filters.ArgIndex,
			&conditions,
			&filters.Args)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if cityStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildSanitizedIlikeCondition(
			cityStr, "COALESCE(edp.venue_city, ep.venue_city)",
			"city", filters.ArgIndex, &conditions, &filters.Args)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if eventUuidsStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildColumnInUuidCondition(
			eventUuidsStr, "edp.event_uuid", filters.ArgIndex, &conditions, &filters.Args)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if orgUuidsStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildColumnInUuidCondition(
			orgUuidsStr, "ep.org_uuid", filters.ArgIndex, &conditions, &filters.Args)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if venueStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildSanitizedIlikeCondition(
			venueStr, "COALESCE(edp.venue_name, ep.venue_name)",
			"venue", filters.ArgIndex, &conditions, &filters.Args)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if venueUuidsStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildColumnInUuidCondition(
			venueUuidsStr, "COALESCE(edp.venue_uuid, ep.venue_uuid)", filters.ArgIndex, &conditions, &filters.Args)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if spaceUuidsStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildColumnInUuidCondition(
			spaceUuidsStr, "COALESCE(edp.space_uuid, ep.space_uuid)", filters.ArgIndex, &conditions, &filters.Args)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if spaceTypesStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildInConditionForStringSlice(
			spaceTypesStr,
			"COALESCE(edp.space_type, ep.space_type) = ANY($%d::text[])",
			filters.ArgIndex, &conditions, &filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	filters.ArgIndex, errBuild = sql_utils.BuildGeoRadiusCondition(
		lonStr, latStr, radiusStr,
		"COALESCE(edp.venue_point, ep.venue_point)",
		filters.ArgIndex, &conditions, &filters.Args)
	if errBuild != nil {
		return filters, errBuild
	}

	filters.ArgIndex, errBuild = sql_utils.BuildContainedInColumnIntRangeCondition(
		ageStr, "ep.min_age", "ep.max_age", filters.ArgIndex, &conditions, &filters.Args)
	if errBuild != nil {
		return filters, errBuild
	}

	filters.ArgIndex, errBuild = sql_utils.BuildPriceCondition(
		priceStr, "ep.price_type", "ep.currency", "ep.min_price", "ep.max_price", "price", filters.ArgIndex, &conditions, &filters.Args)
	if errBuild != nil {
		return filters, errBuild
	}

	filters.ArgIndex, errBuild = sql_utils.BuildBitmaskCondition(
		accessibilityFlagsStr, "edp.space_accessibility_flags", "accessibility", filters.ArgIndex, &conditions, &filters.Args)
	if errBuild != nil {
		return filters, errBuild
	}

	filters.ArgIndex, errBuild = sql_utils.BuildBitmaskCondition(
		visitorInfosStr, "ep.visitor_info_flags", "visitor_infos", filters.ArgIndex, &conditions, &filters.Args)
	if errBuild != nil {
		return filters, errBuild
	}

	if eventTypesStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildJSONArrayIntCondition(
			"or",
			eventTypesStr,
			"types",
			0, // index 0 = type_id
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if genresStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildJSONArrayIntCondition(
			"or",
			genresStr,
			"types",
			1, // index 1 = genre_id
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if tagsStr != "" {
		filters.ArgIndex, errBuild = sql_utils.BuildInConditionForStringSlice(
			tagsStr,
			"tags && $%d::text[]", // Array overlap operator
			filters.ArgIndex,
			&conditions,
			&filters.Args,
		)
		if errBuild != nil {
			return filters, errBuild
		}
	}

	if app.IsValidDateStr(weekStartStr) {
		filters.WeekStart = weekStartStr
	}

	// Join all conditions
	if len(conditions) > 0 {
		filters.ConditionsStr = " AND " + strings.Join(conditions, " AND ")
	}

	// Build limit/offset clause
	filters.LimitClause, filters.ArgIndex, errBuild = sql_utils.BuildLimitOffsetClause(limitStr, offsetStr, filters.ArgIndex, &filters.Args)
	if errBuild != nil {
		return filters, errBuild
	}

	// Geolist
	if geolistRegionStr != "" {
		parts := strings.Split(geolistRegionStr, ",")
		if len(parts) != 3 {
			return filters, errors.New("geolist_region must contain 3 parts")
		}

		countrySlug := parts[0]
		stateSlug := parts[1]
		regionSlug := parts[2]

		pattern := `
			JOIN %s.geolist_country glc ON glc.slug = $%d
			JOIN %s.geolist_state gls ON gls.country_code = glc.code AND gls.slug = $%d
			JOIN %s.geolist_region glr ON glr.country_code = glc.code AND glr.state_code = gls.code AND glr.slug = $%d
		`
		filters.Args = append(filters.Args, countrySlug, stateSlug, regionSlug)
		filters.PortalJoin = fmt.Sprintf(
			pattern,
			h.DbSchema, filters.ArgIndex,
			h.DbSchema, filters.ArgIndex+1,
			h.DbSchema, filters.ArgIndex+2)
		filters.ArgIndex += 3

		filters.PortalConditions = "AND ST_Covers(glr.wkb_geometry, COALESCE(edp.venue_point, ep.venue_point))"
		debugf("filters.PortalConditions: %s", filters.PortalConditions)
	}

	// Portal
	portalUuid, portalUuidExists := GetContextParam(gc, "portal")
	if portalUuidExists {
		if geolistRegionStr != "" {
			return filters, errors.New("portal and geolist_region filters cannot be used together")
		}

		filters.Args = append(filters.Args, portalUuid)
		filters.PortalJoin = fmt.Sprintf("JOIN %s.portal p ON p.uuid = $%d::uuid", h.DbSchema, filters.ArgIndex)
		filters.ArgIndex++

		filters.PortalConditions = fmt.Sprintf(`
			AND (
		        p.wkb_geometry IS NULL
				OR ST_Covers(p.wkb_geometry, COALESCE(edp.venue_point, ep.venue_point))
			)
			AND NOT EXISTS (
    			SELECT 1 FROM %s.portal_org_blacklist b
				WHERE b.portal_uuid = p.uuid AND b.blocked_org_uuid = ep.org_uuid)
			`,
			h.DbSchema)
	}

	return filters, nil
}

func (h *ApiHandler) GetEvents(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-events")
	ctx := gc.Request.Context()
	filters := eventFilters{}

	filters, err := h.buildEventFilters(gc, true)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, err.Error())
		return
	}

	query := app.UranusInstance.SqlGetEventsProjected
	query = strings.Replace(query, "{{date_conditions}}", filters.DateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", filters.ConditionsStr, 1)
	query = strings.Replace(query, "{{limit}}", filters.LimitClause, 1)
	query = strings.Replace(query, "{{portal_join}}", filters.PortalJoin, 1)
	query = strings.Replace(query, "{{portal_conditions}}", filters.PortalConditions, 1)

	debugf("query: %s", query)
	for i, a := range filters.Args {
		fmt.Printf("args[%d] = %#v\n", i, a)
	}

	rows, err := h.DbPool.Query(ctx, query, filters.Args...)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	events := make([]eventResponse, 0)

	for rows.Next() {
		var e eventResponse
		var typesJSON []byte
		err := rows.Scan(
			&e.DateUuid,
			&e.Uuid,
			&e.OrgUuid,
			&e.VenueUuid,
			&e.SpaceUuid,
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
			&e.Summary,
			&e.Categories,
			&typesJSON,
			&e.Languages,
			&e.Tags,
			&e.OrgName,
			&e.ImageUuid,
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
			&e.PriceType,
			&e.VisitorInfoFlags,
		)
		if err != nil {
			apiRequest.InternalServerError()
			return
		}

		e.DateSlug = BuildDateSlug(e.StartDate, e.StartTime)

		// Convert types JSON
		var rawTypes [][]int
		if len(typesJSON) > 0 {
			err := json.Unmarshal(typesJSON, &rawTypes)
			if err != nil {
				apiRequest.InternalServerError()
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

		if e.ImageUuid != nil {
			path := ImageUrl(*e.ImageUuid)
			e.ImagePath = &path
		}

		events = append(events, e)
	}

	if len(events) == 0 {
		response := eventsResponse{
			Events:            events,
			LastEventDateUuid: nil,
			LastEventStartAt:  nil,
		}
		apiRequest.Success(http.StatusOK, response)
		return
	}

	lastEvent := events[len(events)-1]
	lastEventStartAt := lastEvent.StartDate + "T" + lastEvent.StartTime
	lastEventDateUuid := lastEvent.DateUuid
	response := eventsResponse{
		Events:            events,
		LastEventDateUuid: &lastEventDateUuid,
		LastEventStartAt:  &lastEventStartAt,
	}

	apiRequest.Success(http.StatusOK, response)
}

func (h *ApiHandler) GetEventsWeek(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-events-week")
	ctx := gc.Request.Context()

	filters, err := h.buildEventFilters(gc, true)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, err.Error())
		return
	}
	if filters.WeekStart == "" {
		apiRequest.Error(http.StatusBadRequest, "week_start is required")
		return
	}

	query := app.UranusInstance.SqlGetEventsProjectedWeek

	weekEnd, err := computeWeekEnd(filters.WeekStart)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, "invalid week_start")
		return
	}

	query = strings.Replace(query, "{{week_start}}", filters.WeekStart, -1)
	query = strings.Replace(query, "{{week_end}}", weekEnd, -1)
	query = strings.Replace(query, "{{conditions}}", filters.ConditionsStr, 1)
	query = strings.Replace(query, "{{portal_join}}", filters.PortalJoin, 1)
	query = strings.Replace(query, "{{portal_conditions}}", filters.PortalConditions, 1)

	/*
		debugf("filters.ConditionsStr: %v", filters.ConditionsStr)
		debugf("filters.PortalJoin: %v", filters.PortalJoin)
		debugf("filters.PortalConditions: %v", filters.PortalConditions)
		debugf(query)
		debugf("ARGS (%d):\n", len(filters.Args))
		for i, arg := range filters.Args {
			debugf("ARGS[%d]: %#v", i, arg)
		}
	*/

	rows, err := h.DbPool.Query(ctx, query, filters.Args...)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type calendarDayResponse struct {
		EventDay  string          `json:"event_day"`
		Events    json.RawMessage `json:"events"`
		MoreCount int             `json:"more_count"`
	}

	// Preallocate for 7 days (week view)
	days := make([]calendarDayResponse, 0, 7)

	for rows.Next() {
		var (
			day       string
			eventsRaw []byte
			moreCount int
		)

		// SAFE SCAN: only primitives + jsonb as []byte
		if err := rows.Scan(&day, &eventsRaw, &moreCount); err != nil {
			apiRequest.InternalServerError()
			return
		}

		// Normalize NULL JSON → empty array
		if len(eventsRaw) == 0 {
			eventsRaw = []byte("[]")
		}

		// Validate JSON (prevents corrupt upstream SQL)
		if !json.Valid(eventsRaw) {
			apiRequest.InternalServerError()
			return
		}

		days = append(days, calendarDayResponse{
			EventDay:  day,
			Events:    json.RawMessage(eventsRaw),
			MoreCount: moreCount,
		})
	}

	if err := rows.Err(); err != nil {
		apiRequest.InternalServerError()
		return
	}

	// Guarantee stable 7-day output (important for UI calendars)
	response := struct {
		Days []calendarDayResponse `json:"days"`
	}{
		Days: days,
	}

	apiRequest.Success(http.StatusOK, response)
}

func (h *ApiHandler) GetEventTypeSummary(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-events-type-summary")

	filters, err := h.buildEventFilters(gc, false)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, "filter error")
		return
	}

	typeSummary, err := h.loadSummary(gc.Request.Context(), filters, 0)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	genreSummary, err := h.loadSummary(gc.Request.Context(), filters, 1)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	totalQuery := fmt.Sprintf(`
	    SELECT COUNT(DISTINCT edp.event_date_uuid) AS total_count
	    FROM %s.event_date_projection edp
	    JOIN %s.event_projection ep ON ep.event_uuid = edp.event_uuid
	    {{portal_join}}
	    WHERE ep.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled')
	    AND {{date_conditions}}
	    {{conditions}}
	    {{portal_conditions}}`,
		h.DbSchema,
		h.DbSchema,
	)

	totalQuery = strings.Replace(totalQuery, "{{date_conditions}}", filters.DateConditions, 1)
	totalQuery = strings.Replace(totalQuery, "{{conditions}}", filters.ConditionsStr, 1)
	totalQuery = strings.Replace(totalQuery, "{{portal_join}}", filters.PortalJoin, 1)
	totalQuery = strings.Replace(totalQuery, "{{portal_conditions}}", filters.PortalConditions, 1)

	var totalCount int64
	err = h.DbPool.QueryRow(gc.Request.Context(), totalQuery, filters.Args...).Scan(&totalCount)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, gin.H{
		"total_event_count": totalCount,
		"type_summary":      typeSummary,
		"genre_summary":     genreSummary,
	})
}

func (h *ApiHandler) GetEventVenueSummary(gc *gin.Context) {
	// TODO: Use apiRequest
	filters := eventFilters{}

	filters, err := h.buildEventFilters(gc, true)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	conds := []string{"ep.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled'"}

	if filters.DateConditions != "" {
		conds = append(conds, filters.DateConditions)
	}

	if filters.ConditionsStr != "" {
		conds = append(conds, filters.ConditionsStr)
	}

	query := fmt.Sprintf(`
		SELECT jsonb_agg(
			jsonb_build_object(
				'venue_uuid', venue_uuid,
				'venue_name', venue_name,
				'date_count', date_count
			)
			ORDER BY venue_name ASC
		) AS venues
		FROM (
			SELECT
				COALESCE(edp.venue_uuid, ep.venue_uuid) AS venue_uuid,
				COALESCE(edp.venue_name, ep.venue_name) AS venue_name,
				COUNT(edp.event_date_uuid) AS date_count
			FROM %s.event_date_projection edp
			JOIN %s.event_projection ep
			  ON ep.event_uuid = edp.event_uuid
			WHERE %s
			GROUP BY COALESCE(edp.venue_uuid, ep.venue_uuid),
					 COALESCE(edp.venue_name, ep.venue_name)
		) AS venue_counts`,
		h.DbSchema, h.DbSchema, strings.Join(conds, " AND "))

	var jsonResult []byte
	err = h.DbPool.QueryRow(gc.Request.Context(), query, filters.Args...).Scan(&jsonResult)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var venues interface{}
	if err := json.Unmarshal(jsonResult, &venues); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"venue-summary": venues})
}

func (h *ApiHandler) GetEventsGeoJSON(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-events-geojson")
	ctx := gc.Request.Context()
	filters := eventFilters{}

	filters, err := h.buildEventFilters(gc, true)
	if err != nil {
		apiRequest.Error(http.StatusBadRequest, "")
		return
	}

	query := app.UranusInstance.SqlGetEventsGeoJSON
	query = strings.Replace(query, "{{date_conditions}}", filters.DateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", filters.ConditionsStr, 1)
	query = strings.Replace(query, "{{limit}}", filters.LimitClause, 1)
	query = strings.Replace(query, "{{portal_join}}", filters.PortalJoin, 1)
	query = strings.Replace(query, "{{portal_conditions}}", filters.PortalConditions, 1)

	// debugf(query)
	// debugf("ARGS (%d):\n", len(filters.Args))

	for i, arg := range filters.Args {
		fmt.Printf("  $%d = %#v (type %T)\n", i+1, arg, arg)
	}

	rows, err := h.DbPool.Query(ctx, query, filters.Args...)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	type GeoJSONGeometry struct {
		Type        string     `json:"type"`
		Coordinates [2]float64 `json:"coordinates"`
	}

	type GeoJSONFeature struct {
		Type       string                 `json:"type"`
		Geometry   GeoJSONGeometry        `json:"geometry"`
		Properties map[string]interface{} `json:"properties"`
	}

	type GeoJSONFeatureCollection struct {
		Type     string           `json:"type"`
		Features []GeoJSONFeature `json:"features"`
	}

	// Build features

	features := []GeoJSONFeature{}
	totalEvents := 0

	for rows.Next() {

		var venueUuid string
		var venueName string
		var venueCity *string
		var venueCountry *string
		var venueLon *float64
		var venueLat *float64
		var eventCount int

		if err := rows.Scan(
			&venueUuid,
			&venueName,
			&venueCity,
			&venueCountry,
			&venueLon,
			&venueLat,
			&eventCount,
		); err != nil {
			apiRequest.InternalServerError()
			return
		}

		totalEvents += eventCount

		// Skip invalid geometry
		if venueLon == nil || venueLat == nil {
			continue
		}

		feature := GeoJSONFeature{
			Type: "Feature",
			Geometry: GeoJSONGeometry{
				Type: "Point",
				Coordinates: [2]float64{
					*venueLon,
					*venueLat,
				},
			},
			Properties: map[string]interface{}{
				"uuid":        venueUuid,
				"name":        venueName,
				"city":        venueCity,
				"country":     venueCountry,
				"event_count": eventCount,
			},
		}

		features = append(features, feature)
	}

	if err := rows.Err(); err != nil {
		apiRequest.InternalServerError()
		return
	}

	if len(features) == 0 {
		apiRequest.NoContent("no venues found")
		return
	}

	geojson := GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}

	// Response with Metadata
	apiRequest.SetMeta("venue_count", len(features))
	apiRequest.SetMeta("event_count", totalEvents)
	apiRequest.Success(http.StatusOK, geojson)
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

func computeWeekEnd(weekStartStr string) (string, error) {
	const dateLayout = "2006-01-02"
	weekStart, err := time.Parse(dateLayout, weekStartStr)
	if err != nil {
		return "", err
	}
	weekEnd := weekStart.AddDate(0, 0, 6)
	return weekEnd.Format(dateLayout), nil
}

type summaryEntry struct {
	Id    int `json:"id"`
	Count int `json:"count"`
}

func (h *ApiHandler) loadSummary(
	ctx context.Context,
	filters eventFilters,
	jsonIndex int,
) ([]summaryEntry, error) {

	query := fmt.Sprintf(`
		SELECT id, COUNT(*) AS date_count
		FROM (
			SELECT
				edp.event_date_uuid,
				(elem->>%d)::int AS id
			FROM %s.event_date_projection edp
			JOIN %s.event_projection ep
				ON ep.event_uuid = edp.event_uuid
			CROSS JOIN LATERAL jsonb_array_elements(ep.types) AS elem
			{{portal_join}}
			WHERE ep.release_status IN ('released', 'cancelled', 'deferred', 'rescheduled')
			AND {{date_conditions}}
			{{conditions}}
			{{portal_conditions}}
			GROUP BY
				edp.event_date_uuid,
				(elem->>%d)::int
		) grouped
		GROUP BY id
		ORDER BY date_count DESC`,
		jsonIndex,
		h.DbSchema,
		h.DbSchema,
		jsonIndex,
	)

	query = strings.Replace(query, "{{date_conditions}}", filters.DateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", filters.ConditionsStr, 1)
	query = strings.Replace(query, "{{portal_join}}", filters.PortalJoin, 1)
	query = strings.Replace(query, "{{portal_conditions}}", filters.PortalConditions, 1)

	rows, err := h.DbPool.Query(ctx, query, filters.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := make([]summaryEntry, 0)

	for rows.Next() {
		var s summaryEntry
		if err := rows.Scan(&s.Id, &s.Count); err != nil {
			return nil, err
		}
		summary = append(summary, s)
	}

	return summary, rows.Err()
}
