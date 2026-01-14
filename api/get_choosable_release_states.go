package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) GetChoosableReleaseStates(gc *gin.Context) {
	ctx := gc.Request.Context()

	query := fmt.Sprintf(`SELECT status_id, name FROM %s.event_status WHERE iso_639_1 = $1 ORDER BY status_id`, h.Config.DbSchema)

	lang := gc.DefaultQuery("lang", "en")
	rows, err := h.DbPool.Query(ctx, query, lang)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Status struct {
		Id   int    `json:"status_id"`
		Name string `json:"name"`
	}

	var states []Status

	for rows.Next() {
		var status Status
		if err := rows.Scan(&status.Id, &status.Name); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		states = append(states, status)
	}

	if err := rows.Err(); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(http.StatusOK, states)
}
