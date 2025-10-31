package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetOrganizerDashboard(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()

	userId, ok := app.GetCurrentUserOrAbort(gc)
	if !ok {
		return // already sent error response
	}

	startStr, ok := GetContextParam(gc, "start")
	var startDate time.Time
	if ok {
		var err error
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			startDate = time.Now() // fallback on parsing error
		}
	} else {
		startDate = time.Now() // fallback if param missing
	}

	row := pool.QueryRow(ctx, app.Singleton.SqlAdminOrganizerDashboard, userId, startDate)

	var jsonResult []byte
	if err := row.Scan(&jsonResult); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNoContent, gin.H{"error": err.Error()})
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	gc.Data(http.StatusOK, "application/json", jsonResult)
}
