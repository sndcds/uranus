package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableCountries(gc *gin.Context) {
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "en")

	query := fmt.Sprintf(
		`SELECT code, name FROM %s.country WHERE iso_639_1 = $1 ORDER BY name`,
		app.UranusInstance.Config.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Country struct {
		Country     *string `json:"country"`
		CountryName *string `json:"country_name"`
	}

	var countries []Country

	for rows.Next() {
		var country Country
		if err := rows.Scan(
			&country.Country,
			&country.CountryName,
		); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		countries = append(countries, country)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(countries) == 0 {
		gc.JSON(http.StatusOK, []Country{})
		return
	}

	gc.JSON(http.StatusOK, countries)
}
