package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
	"log"
	"net/http"
	"time"
)

func QueryVenueForUser(gc *gin.Context) {

	jsonData, httpStatus, err := queryVenueForUserAsJSON(gc, app.Singleton.MainDbPool)
	if err != nil {
		gc.JSON(httpStatus, gin.H{"error": err.Error()})
		return
	}

	gc.Data(httpStatus, "application/json", jsonData)
}

func queryVenueForUserAsJSON(gc *gin.Context, db *pgxpool.Pool) ([]byte, int, error) {

	// TODO:
	// Check for unknown arguments

	start := time.Now() // Start timer
	ctx := gc.Request.Context()

	userId, ok := GetContextParam(gc, "id")
	fmt.Println("userId", userId)
	if !ok {
		fmt.Println("No user ID provided")
		return nil, http.StatusBadRequest, fmt.Errorf("variable id is required")
	}

	query := app.Singleton.SqlQueryVenueByUser

	// fmt.Println(query)

	rows, err := db.Query(ctx, query, userId)
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

func QueryVenueRightsForUser(gc *gin.Context) {

	jsonData, httpStatus, err := queryVenueRightsForUserAsJSON(gc, app.Singleton.MainDbPool)
	if err != nil {
		gc.JSON(httpStatus, gin.H{"error": err.Error()})
		return
	}

	gc.Data(httpStatus, "application/json", jsonData)
}

func queryVenueRightsForUserAsJSON(gc *gin.Context, db *pgxpool.Pool) ([]byte, int, error) {

	start := time.Now() // Start timer
	ctx := gc.Request.Context()

	userId, err := app.CurrentUserID(gc)
	if userId < 0 {
		return nil, http.StatusUnauthorized, err
	}

	rows, err := db.Query(ctx, app.Singleton.SqlQueryUserVenuesById, userId)
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

	elapsed := time.Since(start)
	milliseconds := int(elapsed.Milliseconds())

	type QueryResponse struct {
		Total   int                      `json:"total"`
		Time    string                   `json:"time"`
		Columns []string                 `json:"columns"`
		Results []map[string]interface{} `json:"events"`
	}

	response := QueryResponse{
		Total:   len(results),
		Columns: columnNames,
		Time:    fmt.Sprintf("%d msec", milliseconds),
		Results: results,
	}

	if response.Total < 1 {
		return nil, http.StatusNoContent, fmt.Errorf("query returned 0 results")
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return jsonData, http.StatusOK, nil
}

func QueryOrganizerDashboardForUser(gc *gin.Context) {

	jsonData, httpStatus, err := queryOrganizerDashboardForUserAsJSON(gc, app.Singleton.MainDbPool)
	if err != nil {
		gc.JSON(httpStatus, gin.H{"error": err.Error()})
		return
	}

	gc.Data(httpStatus, "application/json", jsonData)
}

func queryOrganizerDashboardForUserAsJSON(gc *gin.Context, db *pgxpool.Pool) ([]byte, int, error) {
	start := time.Now()
	ctx := gc.Request.Context()

	userId, err := app.CurrentUserID(gc)
	if userId < 0 {
		return nil, http.StatusUnauthorized, err
	}

	//

	startStr, ok := GetContextParam(gc, "start")
	var startDate time.Time

	if ok {
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			startDate = time.Now() // fallback on parsing error
		}
	} else {
		startDate = time.Now() // fallback if param missing
	}

	row := db.QueryRow(ctx, app.Singleton.SqlQueryUserOrgOverview, userId, startDate)

	var jsonResult []byte
	if err := row.Scan(&jsonResult); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, http.StatusNoContent, fmt.Errorf("no data found")
		}
		return nil, http.StatusInternalServerError, err
	}

	fmt.Println("jsonResult", string(jsonResult))

	elapsed := time.Since(start)
	log.Printf("Query took %s", elapsed)

	return jsonResult, http.StatusOK, nil
}
