package api

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
	"time"
)

func QueryVenueForMap(gc *gin.Context) {

	jsonData, httpStatus, err := queryVenueForMapAsJSON(gc, app.Singleton.MainDb)
	if err != nil {
		gc.JSON(httpStatus, gin.H{"error": err.Error()})
		return
	}

	gc.Data(httpStatus, "application/json", jsonData)
}

func queryVenueForMapAsJSON(gc *gin.Context, db *pgxpool.Pool) ([]byte, int, error) {

	// TODO:
	// Check for unknown arguments

	start := time.Now() // Start timer
	ctx := gc.Request.Context()

	query := app.Singleton.SqlQueryVenueForMap

	fmt.Println(query)

	rows, err := db.Query(ctx, query, "de")
	if err != nil {
		return nil, 500, err
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
			return nil, 500, err
		}

		rowMap := make(map[string]interface{}, len(values))
		for i, col := range columnNames {
			rowMap[col] = values[i]
		}

		results = append(results, rowMap)
	}

	if rows.Err() != nil {
		return nil, 500, rows.Err()
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
		return nil, 200, fmt.Errorf("query returned 0 results")
	} else {
		jsonData, err := json.MarshalIndent(response, "", "  ")
		return jsonData, 200, err
	}
}
