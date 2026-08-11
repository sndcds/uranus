package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/sql_utils"
)

func (h *ApiHandler) GetChoosableVenues(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-choosable-venues")
	ctx := gc.Request.Context()

	nameStr, _ := GetContextParam(gc, "name")
	lat, hasLat := GetContextParamFloat64(gc, "lat")
	lon, hasLon := GetContextParamFloat64(gc, "lon")
	radius, hasRadius := GetContextParamFloat64(gc, "radius")

	var conditions []string
	args := []interface{}{}
	argIndex := 1

	argIndex, errBuild := sql_utils.BuildSanitizedIlikeCondition(nameStr, "name", "name", argIndex, &conditions, &args)
	if errBuild != nil {
		apiRequest.InternalServerError()
		return
	}

	if hasLat && hasLon && hasRadius {
		argIndex, errBuild = sql_utils.BuildGeoRadiusCondition(lon, lat, radius, "point", argIndex, &conditions, &args)
		if errBuild != nil {
			apiRequest.InternalServerError()
			return
		}
	}

	query := fmt.Sprintf("SELECT uuid, scope, name, city, state, country FROM %s.venue", h.DbSchema)
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY LOWER(name) ASC"

	debugf(query)

	rows, err := h.DbPool.Query(ctx, query, args...)
	if err != nil {
		debugf(err.Error())
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	type Venue struct {
		Uuid    string  `json:"uuid"`
		Scope   string  `json:"scope"`
		Name    *string `json:"name"`
		City    *string `json:"city,omitempty"`
		State   *string `json:"state,omitempty"`
		Country *string `json:"country,omitempty"`
	}

	var venues []Venue

	for rows.Next() {
		var venue Venue
		if err := rows.Scan(
			&venue.Uuid,
			&venue.Scope,
			&venue.Name,
			&venue.City,
			&venue.State,
			&venue.Country,
		); err != nil {
			debugf(err.Error())
			apiRequest.DatabaseError()
			return
		}
		venues = append(venues, venue)
	}

	if err := rows.Err(); err != nil {
		debugf(err.Error())
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SetMeta("venue_count", len(venues))
	if len(venues) == 0 {
		apiRequest.Success(http.StatusOK, []Venue{})
		return
	}

	apiRequest.Success(http.StatusOK, venues)
}
