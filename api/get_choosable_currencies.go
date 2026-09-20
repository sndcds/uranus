package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosableCurrencies(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-choosable-currencies")
	ctx := gc.Request.Context()

	onceCurrencies.Do(func() {
		currenciesOptionsQuery = fmt.Sprintf(`
			SELECT code AS id, name FROM %s.currency WHERE iso_639_1 = $1 ORDER BY name`,
			h.DbSchema)
	})

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	rows, err := h.DbPool.Query(ctx, currenciesOptionsQuery, lang)
	if err != nil {
		apiRequest.InternalServerError()
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
			apiRequest.InternalServerError()
			return
		}
		options = append(options, option)
	}

	if err := rows.Err(); err != nil {
		apiRequest.InternalServerError()
		return
	}

	if len(options) == 0 {
		apiRequest.Success(http.StatusOK, []OptionType{})
		return
	}

	apiRequest.SetMeta("currency_count", len(options))
	apiRequest.Success(http.StatusOK, options)
}
