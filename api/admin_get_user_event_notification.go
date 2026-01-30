package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: Only returns notifications for the authenticated user.
// PermissionChecks: Unnecessary.
// Verified: 2026-01-12, Roald

func (h *ApiHandler) AdminGetUserEventNotifications(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	releaseDateDaysLeft := 14
	firstEventDateDaysLeft := 30

	query := app.UranusInstance.SqlAdminGetUserEventNotifications
	rows, err := h.DbPool.Query(ctx, query, userId, releaseDateDaysLeft, firstEventDateDaysLeft)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		notifications = append(notifications, notification)
	}

	if rows.Err() != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": rows.Err().Error()})
		return
	}

	result := map[string]interface{}{
		"notifications": notifications,
		"total_count":   len(notifications),
	}

	gc.JSON(http.StatusOK, result)
}
