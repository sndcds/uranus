package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: Returns notifications for the authenticated user.
// PermissionChecks: Done in PSQL.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetUserEventNotifications(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "admin-get-user-event-notifications")
	userUuid := h.userUuid(gc)

	releaseDateDaysLeft := 14
	firstEventDateDaysLeft := 30

	// For a given user, this query returns all unreleased or draft/review events from organizations
	// they belong to, along with the event’s earliest and latest dates, the number of days until release,
	// and the number of days until the next event occurs. It filters out events that have already ended
	// and applies optional look-ahead windows for release dates or event dates.
	query := app.UranusInstance.SqlAdminGetUserEventNotifications

	rows, err := h.DbPool.Query(ctx, query, userUuid, releaseDateDaysLeft, firstEventDateDaysLeft)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	notifications := []model.UserEventNotification{}

	for rows.Next() {
		var notification model.UserEventNotification
		// Scan all columns returned by your SQL query
		err := rows.Scan(
			&notification.EventId,
			&notification.EventTitle,
			&notification.OrganizationId,
			&notification.OrganizationName,
			&notification.ReleaseDate,
			&notification.ReleaseStatus,
			&notification.EarliestEventDate,
			&notification.LatestEventDate,
			&notification.DaysUntilRelease,
			&notification.DaysUntilEvent,
		)
		if err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}
		notifications = append(notifications, notification)
	}

	if rows.Err() != nil {
		debugf(rows.Err().Error())
		apiRequest.InternalServerError()
		return
	}

	result := map[string]interface{}{
		"notifications": notifications,
		"total_count":   len(notifications),
	}

	apiRequest.Success(http.StatusOK, result, "")
}
