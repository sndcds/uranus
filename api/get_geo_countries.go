package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) GetGeoCountries(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-geo-countries")
	ctx := gc.Request.Context()

	lang := gc.DefaultQuery("lang", "de")

	apiRequest.SetMeta(
		"language",
		lang,
	)

	query := app.UranusInstance.SqlGetGeoCountries

	rows, err := h.DbPool.Query(
		ctx,
		query,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()

	countries := make(
		[]map[string]interface{},
		0,
	)

	for rows.Next() {

		values, err := rows.Values()
		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		country := make(
			map[string]interface{},
			len(values),
		)

		for i, fd := range fieldDescriptions {
			if values[i] != nil {
				country[string(fd.Name)] = values[i]
			}
		}

		countries = append(
			countries,
			country,
		)
	}

	if err := rows.Err(); err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(
		http.StatusOK,
		countries,
	)
}

func (h *ApiHandler) GetGeoCountryStates(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-geo-country-states")
	ctx := gc.Request.Context()

	countrySlug := gc.Param("country_slug")
	if countrySlug == "" {
		apiRequest.Required("parameter country_slug is required")
		return
	}

	apiRequest.SetMeta(
		"country_slug",
		countrySlug,
	)

	lang := gc.DefaultQuery(
		"lang",
		"de",
	)

	apiRequest.SetMeta(
		"language",
		lang,
	)

	query := app.UranusInstance.SqlGetGeoCountryStates

	rows, err := h.DbPool.Query(
		ctx,
		query,
		countrySlug,
	)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()

	states := make(
		[]map[string]interface{},
		0,
	)

	for rows.Next() {
		values, err := rows.Values()

		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		state := make(
			map[string]interface{},
			len(values),
		)

		for i, fd := range fieldDescriptions {
			if values[i] != nil {
				state[string(fd.Name)] = values[i]
			}
		}

		states = append(
			states,
			state,
		)
	}

	if err := rows.Err(); err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(
		http.StatusOK,
		states,
	)
}

func (h *ApiHandler) GetGeoStateRegions(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-geo-state-regions")
	ctx := gc.Request.Context()

	countrySlug := gc.Param("country_slug")
	if countrySlug == "" {
		apiRequest.Required(
			"parameter country_slug is required",
		)
		return
	}

	stateSlug := gc.Param("state_slug")
	if stateSlug == "" {
		apiRequest.Required(
			"parameter state_slug is required",
		)
		return
	}

	apiRequest.SetMeta(
		"country_slug",
		countrySlug,
	)

	apiRequest.SetMeta(
		"state_slug",
		stateSlug,
	)

	lang := gc.DefaultQuery(
		"lang",
		"de",
	)

	apiRequest.SetMeta(
		"language",
		lang,
	)

	query := app.UranusInstance.SqlGetGeoStateRegions

	rows, err := h.DbPool.Query(
		ctx,
		query,
		countrySlug,
		stateSlug,
	)

	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	defer rows.Close()

	fieldDescriptions :=
		rows.FieldDescriptions()

	regions := make(
		[]map[string]interface{},
		0,
	)

	for rows.Next() {

		values, err := rows.Values()

		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		region := make(
			map[string]interface{},
			len(values),
		)

		for i, fd := range fieldDescriptions {

			if values[i] != nil {
				region[string(fd.Name)] = values[i]
			}

		}

		regions = append(
			regions,
			region,
		)
	}

	if err := rows.Err(); err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(
		http.StatusOK,
		regions,
	)
}

func (h *ApiHandler) GetGeoRegion(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-geo-region")
	ctx := gc.Request.Context()

	countrySlug := gc.Param("country_slug")
	stateSlug := gc.Param("state_slug")
	regionSlug := gc.Param("region_slug")

	query := app.UranusInstance.SqlGetGeoRegion

	var (
		countryCode   string
		countryName   string
		countrySlugDB string
		stateCode     string
		stateName     string
		stateSlugDB   string
		regionCode    string
		regionName    string
		regionSlugDB  string
		geometry      string
	)

	err := h.DbPool.QueryRow(
		ctx,
		query,
		countrySlug,
		stateSlug,
		regionSlug,
	).Scan(
		&countryCode,
		&countryName,
		&countrySlugDB,
		&stateCode,
		&stateName,
		&stateSlugDB,
		&regionCode,
		&regionName,
		&regionSlugDB,
		&geometry,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apiRequest.NotFound("geo region not found")
			return
		}
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK,
		gin.H{
			"country": gin.H{
				"code": countryCode,
				"name": countryName,
				"slug": countrySlugDB,
			},
			"state": gin.H{
				"code": stateCode,
				"name": stateName,
				"slug": stateSlugDB,
			},
			"region": gin.H{
				"code":     regionCode,
				"name":     regionName,
				"slug":     regionSlugDB,
				"geometry": json.RawMessage(geometry),
			},
		},
	)
}
