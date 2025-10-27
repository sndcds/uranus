package api_admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func GetAdminEventHandler(gc *gin.Context) {
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	eventID := gc.Param("id")
	if eventID == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	langStr := gc.DefaultQuery("lang", "en")

	query := app.Singleton.SqlGetEvent

	rows, err := pool.Query(ctx, query, eventID, langStr)
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

		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	fmt.Println("JSON:", string(jsonData))

	gc.JSON(http.StatusOK, results)
}
