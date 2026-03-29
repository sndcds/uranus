package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosableCurrencies(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "choosable-currencies")

	onceCurrencies.Do(func() {
		currenciesOptionsQuery = fmt.Sprintf(`
			SELECT code AS id, name FROM %s.currency WHERE iso_639_1 = $1 ORDER BY name`,
			h.DbSchema)
	})

	lang := gc.DefaultQuery("lang", "en")

	rows, err := h.DbPool.Query(ctx, currenciesOptionsQuery, lang)
	if err != nil {
		debugf("Error in GetChoosableCurrencies: %s", err.Error())
		apiRequest.DatabaseError()
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
			debugf("Error in GetChoosableCurrencies: %s", err.Error())
			apiRequest.DatabaseError()
			return
		}
		options = append(options, option)
	}

	if err := rows.Err(); err != nil {
		debugf("Error in GetChoosableCurrencies: %s", err.Error())
		apiRequest.DatabaseError()
		return
	}

	if len(options) == 0 {
		apiRequest.Success(http.StatusOK, []OptionType{}, "")
		return
	}

	apiRequest.Success(http.StatusOK, options, "")
}
