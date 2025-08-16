package api

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
	"net/http"
	"time"
)

func QuerySpacesByVenue(gc *gin.Context) {
	jsonData, httpStatus, err := querySpacesByVenueAsJSON(gc, app.Singleton.MainDbPool)
	if err != nil {
		gc.JSON(httpStatus, gin.H{"error": err.Error()})
		return
	}

	gc.Data(httpStatus, "application/json", jsonData)
}

func querySpacesByVenueAsJSON(gc *gin.Context, db *pgxpool.Pool) ([]byte, int, error) {
	// TODO:
	// Check for unknown arguments

	start := time.Now() // Start timer
	ctx := gc.Request.Context()

	venueId, ok := GetContextParam(gc, "id")
	if !ok {
		// id was not present in the request
		fmt.Println("No venue ID provided")
		return nil, http.StatusBadRequest, fmt.Errorf("valiable id is required")
	}

	query := app.Singleton.SqlQuerySpacesByVenueId
	// fmt.Println(query)

	rows, err := db.Query(ctx, query, venueId)
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

	rowCount := len(results)
	if rowCount < 1 {
		return nil, http.StatusNoContent, fmt.Errorf("nothing found")
	}

	if rows.Err() != nil {
		return nil, http.StatusInternalServerError, rows.Err()
	}

	type QueryResponse struct {
		Total   int                      `json:"total"`
		Time    string                   `json:"time"`
		Columns []string                 `json:"columns"`
		Results []map[string]interface{} `json:"spaces"`
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
