package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) GetChoosableStates(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "choosable-states")

	countryCode := gc.DefaultQuery("country-code", "")
	apiRequest.SetMeta("country_code", countryCode)

	query := fmt.Sprintf(
		`SELECT code, name FROM %s.state WHERE country = $1 ORDER BY name`,
		h.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, countryCode)
	if err != nil {
		apiRequest.DatabaseError()
		return
	}
	defer rows.Close()

	type State struct {
		Code *string `json:"state_code"`
		Name *string `json:"state_name"`
	}

	var states []State

	for rows.Next() {
		var state State
		if err := rows.Scan(
			&state.Code,
			&state.Name,
		); err != nil {
			apiRequest.DatabaseError()
			return
		}
		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SetMeta("state_count", len(states))
	apiRequest.Success(http.StatusOK, states, "")
}
