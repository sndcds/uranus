package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// TODO: Review code

type EventNotification struct {
	EventId           int        `json:"event_id"`
	EventTitle        string     `json:"event_title"`
	OrganizerId       int        `json:"organizer_id"`
	OrganizerName     *string    `json:"organizer_name"`
	ReleaseDate       *time.Time `json:"release_date,omitempty"`
	ReleaseStatusId   int        `json:"release_status_id"`
	ReleaseStatusName *string    `json:"release_status_name"`
	EarliestEventDate *time.Time `json:"earliest_event_date,omitempty"`
	LatestEventDate   *time.Time `json:"latest_event_date,omitempty"`
	DaysUntilRelease  *int       `json:"days_until_release"`
	DaysUntilEvent    *int       `json:"days_until_event"`
}

func (h *ApiHandler) AdminGetUserEventNotification(gc *gin.Context) {
	db := h.DbPool
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")
	langStr := gc.DefaultQuery("lang", "en")

	sql := app.Singleton.SqlAdminGetUserEventNotification

	rows, err := db.Query(ctx, sql, userId, 14, 30, langStr)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	events := []EventNotification{}

	for rows.Next() {
		var e EventNotification
		// Scan all columns returned by your SQL query
		err := rows.Scan(
			&e.EventId,
			&e.EventTitle,
			&e.OrganizerId,
			&e.OrganizerName,
			&e.ReleaseDate,
			&e.ReleaseStatusId,
			&e.ReleaseStatusName,
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
