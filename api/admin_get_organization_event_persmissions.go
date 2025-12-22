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

// TODO: Review code

func (h *ApiHandler) AdminGetOrganizationAddEventPermission(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	userId := gc.GetInt("user-id")

	organizationIdStr := gc.Param("organizationId")
	organizationId, err := strconv.Atoi(organizationIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sql := app.Singleton.SqlAdminGetOrganizationAddEventPermission

	var canAddEvent bool
	err = pool.QueryRow(ctx, sql, userId, organizationId).Scan(&canAddEvent)
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
