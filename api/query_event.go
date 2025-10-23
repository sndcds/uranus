package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/sql"
)

func QueryEvent(gc *gin.Context) {
	jsonData, httpStatus, err := queryEventAsJSON(gc, app.Singleton.MainDbPool)
	if err != nil {
		gc.JSON(httpStatus, gin.H{"error": err.Error()})
		return
	}

	gc.Data(httpStatus, "application/json", jsonData)
}

func queryEventAsJSON(gc *gin.Context, db *pgxpool.Pool) ([]byte, int, error) {
	// TODO:
	// Note on security:
	// This version is still vulnerable to SQL injection if any of the inputs are user-controlled. Safe version using parameterized queries (recommended with database/sql or GORM):

	// TODO:
	// Check for unknown arguments

	start := time.Now() // Start timer
	ctx := gc.Request.Context()

	query := app.Singleton.SqlQueryEvent

	languageStr, _ := GetContextParam(gc, "lang")
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
	genreTypesStr, _ := GetContextParam(gc, "genre_types")
	spaceTypesStr, _ := GetContextParam(gc, "space_types")
	titleStr, _ := GetContextParam(gc, "title")
	cityStr, _ := GetContextParam(gc, "city")
	accessibilityInfosStr, _ := GetContextParam(gc, "accessibility")
	visitorInfosStr, _ := GetContextParam(gc, "visitor_infos")
	ageStr, _ := GetContextParam(gc, "age")
	limitStr, _ := GetContextParam(gc, "limit")
	offsetStr, _ := GetContextParam(gc, "offset")

	// TODO: offset, limit

	eventDateConditions := ""
	var conditions []string
	var args []interface{}
	argIndex := 1 // Postgres uses $1, $2, etc.
	var err error

	if languageStr != "" {
		if !app.IsValidIso639_1(languageStr) {
			return nil, http.StatusInternalServerError, fmt.Errorf("lang format error, %s", languageStr)
		}
	} else {
		languageStr = "de"
	}

	args = append(args, languageStr)
	argIndex++

	if app.IsValidDateStr(startStr) {
		eventDateConditions += "WHERE ed.start >= $" + strconv.Itoa(argIndex)
		args = append(args, startStr)
		argIndex++
	} else if startStr != "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("start %s has invalid format", startStr)
	} else {
		if !hasPast {
			eventDateConditions += "WHERE ed.start >= CURRENT_DATE"
		}
	}

	if app.IsValidDateStr(endStr) {
		eventDateConditions += " AND (ed.end <= $" + strconv.Itoa(argIndex) + " OR ed.start <= $" + strconv.Itoa(argIndex) + ")"
		args = append(args, endStr)
		argIndex++
	} else if endStr != "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("end %s has invalid format", endStr)
	}

	argIndex, err = sql.BuildTimeCondition(timeStr, "start", "time", argIndex, &conditions, &args)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	argIndex, err = sql.BuildSanitizedIlikeCondition(searchStr, "e.description", "search", argIndex, &conditions, &args)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if countryCodesStr != "" {
		format := "v.country_code IN (%s)"
		argIndex, err = sql.BuildInConditionForStringSlice(countryCodesStr, format, "country_codes", argIndex, &conditions, &args)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	if postalCodeStr != "" {
		argIndex, err = sql.BuildLikeConditions(postalCodeStr, "v.postal_code", argIndex, &conditions, &args)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	if eventIdsStr != "" {
		argIndex, err = sql.BuildColumnInIntCondition(eventIdsStr, "e.id", "events", argIndex, &conditions, &args)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	if venueIdsStr != "" {
		argIndex, err = sql.BuildColumnInIntCondition(venueIdsStr, "v.id", "venues", argIndex, &conditions, &args)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	if orgIdsStr != "" {
		argIndex, err = sql.BuildColumnInIntCondition(orgIdsStr, "o.id", "organizers", argIndex, &conditions, &args)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	if spaceIdsStr != "" {
		argIndex, err = sql.BuildColumnInIntCondition(spaceIdsStr, "COALESCE(s.id, es.id)", "spaces", argIndex, &conditions, &args)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	argIndex, err = sql.BuildGeographicRadiusCondition(
		lonStr, latStr, radiusStr, "v.wkb_geometry",
		argIndex, &conditions, &args,
	)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if eventTypesStr != "" {
		format := "EXISTS (SELECT 1 FROM " + app.Singleton.Config.DbSchema + ".event_type_links sub_etl WHERE sub_etl.event_id = e.id AND sub_etl.type_id IN (%s))"
		var err error
		argIndex, err = sql.BuildInCondition(eventTypesStr, format, "event_types", argIndex, &conditions, &args)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	if genreTypesStr != "" {
		format := "EXISTS (SELECT 1 FROM " + app.Singleton.Config.DbSchema + ".event_genre_links sub_egl WHERE sub_egl.event_id = e.id AND sub_egl.type_id IN (%s))"
		var err error
		argIndex, err = sql.BuildInCondition(genreTypesStr, format, "genre_types", argIndex, &conditions, &args)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	if spaceTypesStr != "" {
		format := "COALESCE(s.space_type_id, es.space_type_id) IN (%s)"
		argIndex, err = sql.BuildInCondition(spaceTypesStr, format, "space_types", argIndex, &conditions, &args)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	argIndex, err = sql.BuildSanitizedIlikeCondition(titleStr, "e.title", "title", argIndex, &conditions, &args)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	argIndex, err = sql.BuildSanitizedIlikeCondition(cityStr, "v.city", "city", argIndex, &conditions, &args)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	argIndex, err = sql.BuildContainedInColumnRangeCondition(ageStr, "min_age", "max_age", argIndex, &conditions, &args)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	argIndex, err = sql.BuildBitmaskCondition(accessibilityInfosStr, "ed.accessibility_flags", "accessibility_flags", argIndex, &conditions, &args)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	argIndex, err = sql.BuildBitmaskCondition(visitorInfosStr, "ed.visitor_info_flags", "visitor_info_flags", argIndex, &conditions, &args)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	conditionsStr := ""
	if len(conditions) > 0 {
		conditionsStr = "WHERE " + strings.Join(conditions, " AND ")
	}

	order := "ORDER BY ed.start ASC, e.id ASC"

	// Add LIMIT and OFFSET
	limitClause, argIndex, err := sql.BuildLimitOffsetClause(limitStr, offsetStr, argIndex, &args)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	query = strings.Replace(query, "{{event-date-conditions}}", eventDateConditions, 1)
	query = strings.Replace(query, "{{conditions}}", conditionsStr, 1)
	query = strings.Replace(query, "{{limit}}", limitClause, 1)
	query = strings.Replace(query, "{{order}}", order, 1)

	/*
		fmt.Println(query)
		fmt.Println(args...)
		fmt.Printf("eventDateConditions: %#v\n", eventDateConditions)
		fmt.Printf("conditions: %#v\n", conditions)
		fmt.Printf("args: %d: %#v\n", len(args), args)
	*/

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, http.StatusInternalServerError, err
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
			return nil, http.StatusInternalServerError, err
		}

		rowMap := make(map[string]interface{}, len(values))
		for i, col := range columnNames {
			rowMap[col] = values[i]
		}

		results = append(results, rowMap)
	}

	if rows.Err() != nil {
		return nil, http.StatusInternalServerError, rows.Err()
	}

	type QueryResponse struct {
		Total   int                      `json:"total"`
		Time    string                   `json:"time"`
		Columns []string                 `json:"columns"`
		Results []map[string]interface{} `json:"events"`
	}

	elapsed := time.Since(start)
	milliseconds := int(elapsed.Milliseconds())

	response := QueryResponse{
		Total:   len(results),
		Columns: columnNames,
		Time:    fmt.Sprintf("%d msec", milliseconds),
		Results: results,
	}

	if response.Total < 1 {
		return nil, http.StatusNoContent, fmt.Errorf("query returned 0 results")
	} else {
		jsonData, err := json.MarshalIndent(response, "", "  ")
		return jsonData, http.StatusOK, err
	}
}
