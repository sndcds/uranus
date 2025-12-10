package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetSpaceTypes(gc *gin.Context) {
	pool := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	// Get optional language from query param, default to "en"
	lang := gc.Query("lang")
	if lang == "" {
		lang = "en"
	}

	// SQL query
	sqlQuery := `
		SELECT id, name 
		FROM {{schema}}.space_type 
		WHERE iso_639_1 = $1
		ORDER BY LOWER(name)
	`
	sqlQuery = strings.Replace(sqlQuery, "{{schema}}", app.Singleton.Config.DbSchema, 1)

	// Todo: Support language
	rows, err := pool.Query(ctx, sqlQuery, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var spaceTypes []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		spaceTypes = append(spaceTypes, map[string]interface{}{
			"id":   id,
			"name": strings.TrimSpace(name),
		})
	}

	if len(spaceTypes) == 0 {
		gc.JSON(http.StatusNoContent, gin.H{"message": "no space types found"})
		return
	}

	gc.JSON(http.StatusOK, spaceTypes)
}
