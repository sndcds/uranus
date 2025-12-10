package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableStates(gc *gin.Context) {
	db := app.Singleton.MainDbPool
	ctx := gc.Request.Context()

	countryCode := gc.DefaultQuery("country-code", "")

	sql := fmt.Sprintf(
		`SELECT code, name FROM %s.state WHERE country_code = $1 ORDER BY name`,
		app.Singleton.Config.DbSchema,
	)

	rows, err := db.Query(ctx, sql, countryCode)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type State struct {
		StateCode *string `json:"state_code"`
		StateName *string `json:"state_name"`
	}

	var states []State

	for rows.Next() {
		var state State
		if err := rows.Scan(
			&state.StateCode,
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
