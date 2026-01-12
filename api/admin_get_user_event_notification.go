package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: Only returns notifications for the authenticated user.
// PermissionChecks: Unnecessary.

func (h *ApiHandler) AdminGetUserEventNotification(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	releaseDateDaysLeft := 14
	firstEventDateDaysLeft := 30

	query := app.UranusInstance.SqlAdminGetUserEventNotification
	rows, err := h.DbPool.Query(ctx, query, userId, releaseDateDaysLeft, firstEventDateDaysLeft)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	events := []model.UserEventNotification{}

	for rows.Next() {
		var e model.UserEventNotification
		// Scan all columns returned by your SQL query
		err := rows.Scan(
			&e.EventId,
			&e.EventTitle,
			&e.OrganizationId,
			&e.OrganizationName,
			&e.ReleaseDate,
			&e.ReleaseStatusId,
			&e.EarliestEventDate,
			&e.LatestEventDate,
			&e.DaysUntilRelease,
			&e.DaysUntilEvent,
		)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		events = append(events, e)
	}

	if rows.Err() != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": rows.Err().Error()})
		return
	}

	gc.JSON(http.StatusOK, events)
}
