package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/sql_utils"
)

// TODO: Review code

func (h *ApiHandler) GetEvents(gc *gin.Context) {
	ctx := gc.Request.Context()

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
	eventTypesStr, _ := GetContextParam(gc, "event_types")
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
	eventDateConditionCount := 0
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
		eventDateConditionCount++
		args = append(args, startStr)
		argIndex++
	} else if startStr != "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("start %s has invalid format", startStr)})
	} else {
		if !hasPast {
			eventDateConditions += "WHERE ed.start_date >= CURRENT_DATE"
			eventDateConditionCount++
		}
	}

	if app.IsValidDateStr(endStr) {
		if eventDateConditionCount > 0 {
			eventDateConditions += " AND "
		}

		eventDateConditions += "(ed.end_date <= $" + strconv.Itoa(argIndex) + " OR ed.start_date <= $" + strconv.Itoa(argIndex) + ")"
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

	if eventTypesStr != "" {
		if eventDateConditionCount > 0 {
			eventDateConditions += " AND "
		}

		ids, err := app.ParseIntSliceCSV(eventTypesStr)
		if err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		eventDateConditions +=
			"EXISTS (SELECT 1 FROM " + h.DbSchema +
				".event_type_link sub_etl WHERE sub_etl.event_id = e.id AND sub_etl.type_id = ANY($" +
				strconv.Itoa(argIndex) + "))"

		args = append(args, ids)
		argIndex++
	}

	/* TODO: Handle event types and genres, must be refactored!
	if genreTypesStr != "" {
		format := "EXISTS (SELECT 1 FROM " + h.DbSchema + ".event_genre_link sub_egl WHERE sub_egl.event_id = e.id AND sub_egl.type_id IN (%s))"
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

	if isTypeSummaryMode {
		row := h.DbPool.QueryRow(ctx, query, args...)

		var jsonResult []byte
		err := row.Scan(&jsonResult)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		gc.Data(http.StatusOK, "application/json", jsonResult)
		return
	}

	var limitClause string
	limitClause, argIndex, err = sql_utils.BuildLimitOffsetClause(limitStr, offsetStr, argIndex, &args)
	if err != nil {

		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	query = strings.Replace(query, "{{limit}}", limitClause, 1)

	order := "ORDER BY (ed.start_date + COALESCE(ed.start_time, '00:00:00'::time)) ASC, e.id ASC"
	query = strings.Replace(query, "{{order}}", order, 1)

	rows, err := h.DbPool.Query(ctx, query, args...)
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

		// Add extra property image_path
		imageId := rowMap["image_id"]
		if imageId == nil {
			rowMap["image_path"] = nil
		} else {
			rowMap["image_path"] = fmt.Sprintf(
				"%s/api/image/%v",
				app.Singleton.Config.BaseApiUrl,
				imageId,
			)
		}

		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make(map[string]interface{})
	response["events"] = results

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, response)
}
