package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableStates(gc *gin.Context) {
	ctx := gc.Request.Context()

	countryCode := gc.DefaultQuery("country-code", "")

	query := fmt.Sprintf(
		`SELECT code, name FROM %s.state WHERE country = $1 ORDER BY name`,
		app.UranusInstance.Config.DbSchema,
	)

	rows, err := h.DbPool.Query(ctx, query, countryCode)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type State struct {
		State     *string `json:"state"`
		StateName *string `json:"state_name"`
	}

	var states []State

	for rows.Next() {
		var state State
		if err := rows.Scan(
			&state.State,
			&state.StateName,
		); err != nil {
			fmt.Println(err.Error())
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(states) == 0 {
		gc.JSON(http.StatusOK, []State{})
		return
	}

	gc.JSON(http.StatusOK, states)
}
