package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableCurrencies(gc *gin.Context) {
	ctx := gc.Request.Context()

	onceCurrencies.Do(func() {
		currenciesOptionsQuery = fmt.Sprintf(`
			SELECT code AS id, name FROM %s.currency WHERE iso_639_1 = $1 ORDER BY name`,
			h.Config.DbSchema)
	})

	lang := gc.DefaultQuery("lang", "en")

	rows, err := h.DbPool.Query(ctx, currenciesOptionsQuery, lang)
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
