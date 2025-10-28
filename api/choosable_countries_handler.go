package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func ChoosableCountriesHandler(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	langStr := gc.DefaultQuery("lang", "en")

	sql := fmt.Sprintf(
		`SELECT code, name FROM %s.country WHERE iso_639_1 = $1 ORDER BY name`,
		app.Singleton.Config.DbSchema,
	)

	rows, err := db.Query(ctx, sql, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Country struct {
		CountryCode *string `json:"country_code"`
		CountryName *string `json:"country_name"`
	}

	var countries []Country

	for rows.Next() {
		var country Country
		if err := rows.Scan(
			&country.CountryCode,
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
