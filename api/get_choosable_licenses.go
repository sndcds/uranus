package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) GetChoosableLicenses(gc *gin.Context) {
	ctx := gc.Request.Context()
	// userId := gc.GetInt("user-id")

	langStr := gc.DefaultQuery("lang", "en")
	useLongName := gc.DefaultQuery("long", "false") == "true"

	var query string
	if useLongName {
		query = fmt.Sprintf(
			`SELECT license_id, name FROM %s.license_type WHERE iso_639_1 = $1 ORDER BY name`,
			h.DbSchema,
		)
	} else {
		query = fmt.Sprintf(
			`SELECT license_id, short_name FROM %s.license_type WHERE iso_639_1 = $1 ORDER BY short_name`,
			h.DbSchema,
		)
	}

	rows, err := h.DbPool.Query(ctx, query, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Option struct {
		Id   int64   `json:"id"`
		Name *string `json:"name"`
	}

	var options []Option

	for rows.Next() {
		var option Option
		if err := rows.Scan(
			&option.Id,
			&option.Name,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		options = append(options, option)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(options) == 0 {
		gc.JSON(http.StatusOK, []Option{})
		return
	}

	gc.JSON(http.StatusOK, options)
}
