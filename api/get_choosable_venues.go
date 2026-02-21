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
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "choosable-venues")

	nameStr, _ := GetContextParam(gc, "name")
	latStr, _ := GetContextParam(gc, "lat")
	lonStr, _ := GetContextParam(gc, "lon")
	radiusStr, _ := GetContextParam(gc, "radius")

	var conditions []string
	args := []interface{}{}
	argIndex := 1

	argIndex, errBuild := sql_utils.BuildSanitizedIlikeCondition(nameStr, "name", "name", argIndex, &conditions, &args)
	if errBuild != nil {
		apiRequest.InternalServerError()
		return
	}

	argIndex, errBuild = sql_utils.BuildGeoRadiusCondition(lonStr, latStr, radiusStr, "geo_pos", argIndex, &conditions, &args)
	if errBuild != nil {
		apiRequest.InternalServerError()
		return
	}

	debugf("argIndex = %d", argIndex)
	debugf("len(conditions) = %d", len(conditions))
	query := fmt.Sprintf("SELECT id, name, city, state, country FROM %s.venue", h.DbSchema)
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY LOWER(name) ASC"

	fmt.Println(query)

	rows, err := h.DbPool.Query(ctx, query, args...)
	if err != nil {
		debugf("1")
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	type Venue struct {
		Id      int64   `json:"id"`
		Name    *string `json:"name"`
		City    *string `json:"city,omitempty"`
		State   *string `json:"state,omitempty"`
		Country *string `json:"country,omitempty"`
	}

	var venues []Venue

	for rows.Next() {
		var venue Venue
		if err := rows.Scan(
			&venue.Id,
			&venue.Name,
			&venue.City,
			&venue.State,
			&venue.Country,
		); err != nil {
			debugf("2")
			apiRequest.DatabaseError()
			return
		}
		venues = append(venues, venue)
	}

	if err := rows.Err(); err != nil {
		debugf("3")
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SetMeta("venue_count", len(venues))
	if len(venues) == 0 {
		apiRequest.Success(http.StatusOK, []Venue{}, "")
		return
	}

	apiRequest.Success(http.StatusOK, venues, "")
}
