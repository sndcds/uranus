package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: Returns notifications about events for the authenticated user.
// PermissionChecks: Done in PSQL.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetUserEventNotifications(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-user-event-notifications")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}

	eventLookaheadDays, err := strconv.Atoi(gc.DefaultQuery("event-lookahead-days", "14"))
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	apiRequest.SetMeta("event-lookahead-days", eventLookaheadDays)

	query := app.UranusInstance.SqlAdminGetUserEventNotifications

	rows, err := h.DbPool.Query(ctx, query, userUuid, orgUuid, eventLookaheadDays)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return

	}

	defer rows.Close()

	notifications, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.UserEventNotification])
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return

	}

	type Response struct {
		Notifications []model.UserEventNotification `json:"notifications"`
		TotalCount    int                           `json:"total_count"`
	}

	result := Response{
		Notifications: notifications,
		TotalCount:    len(notifications),
	}

	apiRequest.Success(http.StatusOK, result)
}
