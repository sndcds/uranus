package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableEventUrlTypes(gc *gin.Context) {
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")

	query := fmt.Sprintf(
		`SELECT type_id, type_name FROM %s.url_type WHERE context = 'event' AND iso_639_1 = $1 ORDER BY LOWER(type_name)`,
		h.Config.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type UrlType struct {
		Id   *string `json:"id"`
		Name *string `json:"name"`
	}

	var urlTypes []UrlType

	for rows.Next() {
		var urlType UrlType
		if err := rows.Scan(
			&urlType.Id,
			&urlType.Name,
		); err != nil {
			fmt.Println(err.Error())
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		urlTypes = append(urlTypes, urlType)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(urlTypes) == 0 {
		gc.JSON(http.StatusOK, []UrlType{})
		return
	}

	gc.JSON(http.StatusOK, urlTypes)
}
