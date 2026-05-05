package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/sql_utils"
)

func (h *ApiHandler) GetChoosableOrgs(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "choosable-orgs")
	ctx := gc.Request.Context()

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

	argIndex, errBuild = sql_utils.BuildGeoRadiusCondition(lonStr, latStr, radiusStr, "point", argIndex, &conditions, &args)
	if errBuild != nil {
		apiRequest.InternalServerError()
		return
	}

	query := fmt.Sprintf("SELECT uuid, name, city, state, country FROM %s.organization", h.DbSchema)
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY LOWER(name) ASC"

	rows, err := h.DbPool.Query(ctx, query, args...)
	if err != nil {
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	type Org struct {
		Uuid    string  `json:"uuid"`
		Name    *string `json:"name"`
		City    *string `json:"city,omitempty"`
		State   *string `json:"state,omitempty"`
		Country *string `json:"country,omitempty"`
	}

	var orgs []Org

	for rows.Next() {
		var org Org
		if err := rows.Scan(
			&org.Uuid,
			&org.Name,
			&org.City,
			&org.State,
			&org.Country,
		); err != nil {
			apiRequest.DatabaseError()
			return
		}
		orgs = append(orgs, org)
	}

	if err := rows.Err(); err != nil {
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SetMeta("venue_count", len(orgs))
	if len(orgs) == 0 {
		apiRequest.Success(http.StatusOK, []Org{}, "")
		return
	}

	apiRequest.Success(http.StatusOK, orgs, "")
}
