package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/sql_utils"
)

// TODO: Review code

func (h *ApiHandler) GetEvents(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	var query string
	var isTypeSummaryMode bool
	modeStr := gc.Query("mode") // "" if not provided
	switch modeStr {
	case "", "basic", "geometry":
		query = app.Singleton.SqlGetEventsBasic
	case "extended":
		query = app.Singleton.SqlGetEventsExtended
	case "detailed":
		query = app.Singleton.SqlGetEventsDetailed
	case "type-summary":
		query = app.Singleton.SqlGetEventsTypeSummary
		isTypeSummaryMode = true
	default:
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Unknown mode %s", modeStr)})
		return
	}

	// TODO:
	// Note on security:
	// This version is still vulnerable to SQL injection if any of the inputs are user-controlled. Safe version using parameterized queries (recommended with database/sql_utils or GORM):

	// TODO:
	// Check for unknown arguments

	langStr, _ := GetContextParam(gc, "lang")
	_, hasPast := GetContextParam(gc, "past")
	startStr, _ := GetContextParam(gc, "start")
	endStr, _ := GetContextParam(gc, "end")
	timeStr, _ := GetContextParam(gc, "time")
	searchStr, _ := GetContextParam(gc, "search")
	eventIdsStr, _ := GetContextParam(gc, "events")
	venueIdsStr, _ := GetContextParam(gc, "venues")
	spaceIdsStr, _ := GetContextParam(gc, "spaces")
	orgIdsStr, _ := GetContextParam(gc, "organizers")
	countryCodesStr, _ := GetContextParam(gc, "countries")
	// stateCode := GetContextParam(gc, "state_code")
	postalCodeStr, _ := GetContextParam(gc, "postal_code")
	// buildingLevelCodeStr := GetContextParam(gc, "building_level")
	// buildingMinLevelCodeStr := GetContextParam(gc, "building_min_level")
	// buildingMaxLevelCodeStr := GetContextParam(gc, "building_max_level")
	// spaceMinCapacityCodeStr := GetContextParam(gc, "space_min_capacity")
	// spaceMaxCapacityCodeStr := GetContextParam(gc, "space_max_capacity")
	// spaceMinSeatsCodeStr := GetContextParam(gc, "space_min_seats")
	// spaceMaxSeatsCodeStr := GetContextParam(gc, "space_max_seats")
	lonStr, _ := GetContextParam(gc, "lon")
	latStr, _ := GetContextParam(gc, "lat")
	radiusStr, _ := GetContextParam(gc, "radius")
	// eventTypesStr, _ := GetContextParam(gc, "event_types") // Todo: must be refactored
	// genreTypesStr, _ := GetContextParam(gc, "genre_types") // Todo: must be refactored
	spaceTypesStr, _ := GetContextParam(gc, "space_types")
	titleStr, _ := GetContextParam(gc, "title")
	cityStr, _ := GetContextParam(gc, "city")
	accessibilityInfosStr, _ := GetContextParam(gc, "accessibility")
	visitorInfosStr, _ := GetContextParam(gc, "visitor_infos")
	ageStr, _ := GetContextParam(gc, "age")
	offsetStr, _ := GetContextParam(gc, "offset")
	limitStr, _ := GetContextParam(gc, "limit")

	eventDateConditions := ""
	var conditions []string
	var args []interface{}
	argIndex := 1 // Postgres uses $1, $2, etc.
	var err error

	if langStr != "" {
		// TODO: Check available languages
		if !app.IsValidIso639_1(langStr) {
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("lang format error %v", err)})
		}
	} else {
		langStr = "en"
	}

	args = append(args, langStr)
	argIndex++

	if app.IsValidDateStr(startStr) {
		eventDateConditions += "WHERE ed.start_date >= $" + strconv.Itoa(argIndex)
		args = append(args, startStr)
		argIndex++
	} else if startStr != "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("start %s has invalid format", startStr)})
	} else {
		if !hasPast {
			eventDateConditions += "WHERE ed.start_date >= CURRENT_DATE"
		}
	}

	if app.IsValidDateStr(endStr) {
		eventDateConditions += " AND (ed.end_date <= $" + strconv.Itoa(argIndex) + " OR ed.start_date <= $" + strconv.Itoa(argIndex) + ")"
		args = append(args, endStr)
		argIndex++
	} else if endStr != "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("end %s has invalid format", endStr)})
	}

	argIndex, err = sql_utils.BuildTimeCondition(timeStr, "start", "time", argIndex, &conditions, &args)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	argIndex, err = sql_utils.BuildSanitizedSearchCondition(searchStr, "e.search_text", "search", argIndex, &conditions, &args)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	if countryCodesStr != "" {
		format := "vd.venue_country_code IN (%s)"
		argIndex, err = sql_utils.BuildInConditionForStringSlice(countryCodesStr, format, "country_codes", argIndex, &conditions, &args)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}

	if postalCodeStr != "" {
		argIndex, err = sql_utils.BuildLikeConditions(postalCodeStr, "vd.postal_code", argIndex, &conditions, &args)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}

	if eventIdsStr != "" {
		argIndex, err = sql_utils.BuildColumnInIntCondition(eventIdsStr, "e.id", "events", argIndex, &conditions, &args)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}

	if venueIdsStr != "" {
		argIndex, err = sql_utils.BuildColumnInIntCondition(venueIdsStr, "vd.venue_id", "venues", argIndex, &conditions, &args)
		if err != nil {

			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}

	if orgIdsStr != "" {
		argIndex, err = sql_utils.BuildColumnInIntCondition(orgIdsStr, "o.id", "organizers", argIndex, &conditions, &args)
		if err != nil {

			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}

	if spaceIdsStr != "" {
		argIndex, err = sql_utils.BuildColumnInIntCondition(spaceIdsStr, "COALESCE(s.id, es.id)", "spaces", argIndex, &conditions, &args)
		if err != nil {

			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}

	argIndex, err = sql_utils.BuildGeographicRadiusCondition(
		lonStr, latStr, radiusStr, "vd.venue_wkb_geometry",
		argIndex, &conditions, &args,
	)

	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	/* TODO: Handle event types and genres, must be refactored!
	if eventTypesStr != "" {
		format := "EXISTS (SELECT 1 FROM " + app.Singleton.Config.DbSchema + ".event_type_link sub_etl WHERE sub_etl.event_id = e.id AND sub_etl.type_id IN (%s))"
		var err error
		argIndex, err = sql_utils.BuildInCondition(eventTypesStr, format, "event_types", argIndex, &conditions, &args)
		if err != nil {

			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}

	if genreTypesStr != "" {
		format := "EXISTS (SELECT 1 FROM " + app.Singleton.Config.DbSchema + ".event_genre_link sub_egl WHERE sub_egl.event_id = e.id AND sub_egl.type_id IN (%s))"
		var err error
		argIndex, err = sql_utils.BuildInCondition(genreTypesStr, format, "genre_types", argIndex, &conditions, &args)
		if err != nil {

			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}
	*/

	if spaceTypesStr != "" {
		format := "COALESCE(s.space_type_id, es.space_type_id) IN (%s)"
		argIndex, err = sql_utils.BuildInCondition(spaceTypesStr, format, "space_types", argIndex, &conditions, &args)
		if err != nil {

			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}

	argIndex, err = sql_utils.BuildSanitizedIlikeCondition(titleStr, "e.title", "title", argIndex, &conditions, &args)
	if err != nil {

		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	argIndex, err = sql_utils.BuildSanitizedIlikeCondition(cityStr, "venue_city", "city", argIndex, &conditions, &args)
	if err != nil {

		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	argIndex, err = sql_utils.BuildContainedInColumnRangeCondition(ageStr, "min_age", "max_age", argIndex, &conditions, &args)
	if err != nil {

		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	argIndex, err = sql_utils.BuildBitmaskCondition(accessibilityInfosStr, "ed.accessibility_flags", "accessibility_flags", argIndex, &conditions, &args)
	if err != nil {

		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	argIndex, err = sql_utils.BuildBitmaskCondition(visitorInfosStr, "ed.visitor_info_flags", "visitor_info_flags", argIndex, &conditions, &args)
	if err != nil {

		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	conditionsStr := ""
	if len(conditions) > 0 {
		conditionsStr = "WHERE " + strings.Join(conditions, " AND ")
	}

	query = strings.Replace(query, "{{event-date-conditions}}", eventDateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", conditionsStr, 1)

	// Add LIMIT and OFFSET
	if isTypeSummaryMode {
		query = strings.Replace(query, "{{limit}}", "", 1)
	} else {
		var limitClause string
		limitClause, argIndex, err = sql_utils.BuildLimitOffsetClause(limitStr, offsetStr, argIndex, &args)
		if err != nil {

			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		query = strings.Replace(query, "{{limit}}", limitClause, 1)
	}

	order := "ORDER BY (ed.start_date + COALESCE(ed.start_time, '00:00:00'::time)) ASC, e.id ASC"
	query = strings.Replace(query, "{{order}}", order, 1)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columnNames[i] = string(fd.Name)
	}

	var results []map[string]interface{}

	// Define a struct matching the JSON structure of event_types
	type EventType struct {
		TypeID    int    `json:"type_id"`
		TypeName  string `json:"type_name"`
		GenreID   int    `json:"genre_id"`
		GenreName string `json:"genre_name"`
	}

	// Map to accumulate counts: type_id -> {name, count}
	type TypeCountEntry struct {
		Id    int
		Name  string
		Count int
	}

	type VenueSummary struct {
		ID             int    `json:"id"`
		Name           string `json:"name"`
		City           string `json:"city"`
		EventDateCount int    `json:"event_date_count"`
	}

	type OrganizerSummary struct {
		ID             int    `json:"id"`
		Name           string `json:"name"`
		EventDateCount int    `json:"event_date_count"`
	}

	typeCount := make(map[int]TypeCountEntry)
	venueMap := make(map[int]*VenueSummary)
	organizerMap := make(map[int]*OrganizerSummary)

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rowMap := make(map[string]interface{}, len(values))
		for i, col := range columnNames {
			rowMap[col] = values[i]
		}

		if !isTypeSummaryMode {
			// Add extra property image_path
			imageID := rowMap["image_id"]
			if imageID == nil {
				rowMap["image_path"] = nil // or "" if you prefer
			} else {
				rowMap["image_path"] = fmt.Sprintf(
					"%s/api/image/%v",
					app.Singleton.Config.BaseApiUrl,
					imageID,
				)
			}
		}

		// Accumulate event type counts
		var eventTypes []EventType
		switch v := rowMap["event_types"].(type) {
		case nil:
			// no event types to include in summary
		case string:
			if err := json.Unmarshal([]byte(v), &eventTypes); err != nil {
				fmt.Println("json.Unmarshal error:", err) // TODO: What should happen in this case?
			}
		case []byte:
			if err := json.Unmarshal(v, &eventTypes); err != nil {
				fmt.Println("json.Unmarshal error:", err) // TODO: What should happen in this case?
			}
		case []interface{}:
			// Already decoded array of maps
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					typeID, _ := m["type_id"].(float64) // convert float64 -> int
					typeName, _ := m["type_name"].(string)
					eventTypes = append(eventTypes, EventType{
						TypeID:   int(typeID),
						TypeName: typeName,
					})
				}
			}
		default:
			// TODO: Is this an error?
			// fmt.Printf("unexpected type for event_types: %T\n", v)
		}

		for _, et := range eventTypes {
			entry := typeCount[et.TypeID]
			entry.Name = et.TypeName
			entry.Count++
			typeCount[et.TypeID] = entry
		}

		// Organizer summary
		organizerId, ok := app.ToInt(rowMap["organizer_id"])
		if ok {
			organizerName, _ := rowMap["organizer_name"].(string)

			// Initialize organizer summary if not yet present
			if _, exists := organizerMap[organizerId]; !exists {
				organizerMap[organizerId] = &OrganizerSummary{
					ID:             organizerId,
					Name:           organizerName,
					EventDateCount: 0,
				}
			}

			// Increment event count for this organizer
			organizerMap[organizerId].EventDateCount++
		}

		// Venue summary
		venueId, ok := app.ToInt(rowMap["venue_id"])
		if ok {
			venueName, _ := rowMap["venue_name"].(string)
			venueCity, _ := rowMap["venue_city"].(string)

			// Initialize venue summary if not yet present
			if _, exists := venueMap[venueId]; !exists {
				venueMap[venueId] = &VenueSummary{
					ID:             venueId,
					Name:           venueName,
					City:           venueCity,
					EventDateCount: 0,
				}
			}

			// Increment event count for this venue
			venueMap[venueId].EventDateCount++
		}

		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert typeCount map to a slice for JSON output
	var typeSummary []map[string]interface{}
	for id, entry := range typeCount {
		typeSummary = append(typeSummary, map[string]interface{}{
			"type_id":   id,
			"type_name": entry.Name,
			"count":     entry.Count,
		})
	}

	sort.Slice(typeSummary, func(i, j int) bool {
		nameI := typeSummary[i]["type_name"].(string)
		nameJ := typeSummary[j]["type_name"].(string)
		return nameI < nameJ
	})

	// Total number of events
	totalEvents := len(results)

	// Organizer summary
	organizerSummary := make([]OrganizerSummary, 0, len(organizerMap))
	for _, summary := range organizerMap {
		organizerSummary = append(organizerSummary, *summary)
	}

	// Venues summary
	venuesSummary := make([]VenueSummary, 0, len(venueMap))
	for _, summary := range venueMap {
		venuesSummary = append(venuesSummary, *summary)
	}

	sort.Slice(venuesSummary, func(i, j int) bool {
		return venuesSummary[i].Name < venuesSummary[j].Name
	})

	// TODO: Check if query does unneccessary work!
	response := make(map[string]interface{})
	if isTypeSummaryMode {
		response["total"] = totalEvents
		response["type_summary"] = typeSummary
		response["organizer_summary"] = organizerSummary
		response["venues_summary"] = venuesSummary

	} else {
		response["events"] = results
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return combined JSON
	gc.JSON(http.StatusOK, response)
}
