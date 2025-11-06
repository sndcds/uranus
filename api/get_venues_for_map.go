package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetGeojsonVenues(gc *gin.Context) {
	db := h.DbPool
	ctx := gc.Request.Context()

	sql := app.Singleton.SqlGetGeojsonVenues
	// TODO: languageStr, default "en"
	rows, err := db.Query(ctx, sql, "en")
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columnNames[i] = string(fd.Name)
	}

	// Iterate over rows and build JSON
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

	if len(results) == 0 {
		gc.JSON(http.StatusNoContent, gin.H{"message": "query returned 0 results"})
		return
	}

	gc.IndentedJSON(http.StatusOK, results)
}
