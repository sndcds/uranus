package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) GetChoosablePriceTypes(gc *gin.Context) {
	ctx := gc.Request.Context()

	oncePriceTypes.Do(func() {
		priceTypesOptionsQuery = fmt.Sprintf(`
			SELECT type_id AS id, name FROM %s.price_type WHERE iso_639_1 = $1
			ORDER BY CASE WHEN type_id = 0 THEN 0 ELSE 1 END, name`,
			h.DbSchema)
	})

	lang := gc.DefaultQuery("lang", "en")

	rows, err := h.DbPool.Query(ctx, priceTypesOptionsQuery, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type OptionType struct {
		Id   *string `json:"id"`
		Name *string `json:"name"`
	}

	var options []OptionType

	for rows.Next() {
		var option OptionType
		if err := rows.Scan(
			&option.Id,
			&option.Name,
		); err != nil {
			fmt.Println(err.Error())
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
		gc.JSON(http.StatusOK, []OptionType{})
		return
	}

	gc.JSON(http.StatusOK, options)
}
