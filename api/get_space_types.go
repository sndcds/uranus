package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) GetChoosableSpaceTypes(gc *gin.Context) {
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")

	// SQL query
	query := fmt.Sprintf(`
		SELECT type_id, name 
		FROM %s.space_type 
		WHERE iso_639_1 = $1
		ORDER BY LOWER(name)`, h.DbSchema)
	fmt.Println(query)
	rows, err := h.DbPool.Query(ctx, query, lang)
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
