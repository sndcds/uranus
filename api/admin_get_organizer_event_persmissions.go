package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetOrganizerAddEventPermission(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := gc.GetInt("user-id")

	organizerIdStr := gc.Param("organizerId")
	organizerId, err := strconv.Atoi(organizerIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sql := app.Singleton.SqlAdminGetOrganizerAddEventPermission

	var canAddEvent bool
	err = pool.QueryRow(ctx, sql, userId, organizerId).Scan(&canAddEvent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			gc.JSON(http.StatusNotFound, gin.H{"error": "no permissions found"})
			return
		}
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("query failed: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"can_add_event": canAddEvent})
}
