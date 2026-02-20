package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosableCountries(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "choosable-countries")

	lang := gc.DefaultQuery("lang", "en")
	apiRequest.SetMeta("language", lang)

	query := fmt.Sprintf(
		`SELECT code, name FROM %s.country WHERE iso_639_1 = $1 ORDER BY name`,
		h.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	type Country struct {
		Code *string `json:"country_code"`
		Name *string `json:"country_name"`
	}

	var countries []Country

	for rows.Next() {
		var country Country
		err := rows.Scan(&country.Code, &country.Name)
		if err != nil {
			apiRequest.DatabaseError()
			return
		}
		countries = append(countries, country)
	}

	if err := rows.Err(); err != nil {
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SetMeta("country_count", len(countries))
	apiRequest.Success(http.StatusOK, countries, "")
}
