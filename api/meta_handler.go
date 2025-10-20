package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func GetMetaHandler(gc *gin.Context) {
	modeStr := gc.Param("mode")

	switch modeStr {
	case "genres":
		fetchEventGenres(gc)
		break
	default:
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("API route does not exist: %s", gc.FullPath()),
		})
	}
}

func fetchEventGenres(gc *gin.Context) {
	eventType, exists := GetContextParameterAsInt(gc, "event-type")
	if exists {
		fetchEventGenresByEventType(gc, eventType)
	} else {
		fetchAllEventGenres(gc)
	}
}

func fetchAllEventGenres(gc *gin.Context) {
	ctx := gc.Request.Context()
	db := app.Singleton.MainDbPool
	sql := app.Singleton.SqlGetMetaGenres

	// Get language from Gin context (e.g. "de", "en")
	languageVal, exists := GetContextParam(gc, "language")
	if !exists {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "variable language is required"})
		return
	}

	rows, err := db.Query(ctx, sql, languageVal)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Prepare to collect rows
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

	if rows.Err() != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": rows.Err().Error()})
		return
	}

	// Build the full response object with "genres" and "language"
	responseObj := map[string]interface{}{
		"api":          app.Singleton.APIName,
		"version":      app.Singleton.APIVersion,
		"event-genres": results,
		"language":     languageVal,
	}

	// Encode and send result
	jsonBytes, err := json.Marshal(responseObj)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.Data(http.StatusOK, "application/json", jsonBytes)
}

func fetchEventGenresByEventType(gc *gin.Context, eventType int) {
	ctx := gc.Request.Context()
	db := app.Singleton.MainDbPool
	sql := app.Singleton.SqlGetMetaGenresByEventType

	languageVal, exists := GetContextParam(gc, "language")
	if !exists {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "variable language is required"})
		return
	}

	rows, err := db.Query(ctx, sql, eventType, languageVal)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Prepare to collect rows
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

	if rows.Err() != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": rows.Err().Error()})
		return
	}

	// Build the full response object with "genres" and "language"
	responseObj := map[string]interface{}{
		"api":          app.Singleton.APIName,
		"version":      app.Singleton.APIVersion,
		"event-genres": results,
		"language":     languageVal,
		"event-type":   eventType,
	}

	// Encode and send result
	jsonBytes, err := json.Marshal(responseObj)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.Data(http.StatusOK, "application/json", jsonBytes)
}
