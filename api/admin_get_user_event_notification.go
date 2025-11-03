package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type EventNotification struct {
	EventId           int        `json:"event_id"`
	EventTitle        string     `json:"event_title"`
	OrganizerId       int        `json:"organizer_id"`
	OrganizerName     *string    `json:"organizer_name"`
	ReleaseDate       *time.Time `json:"release_date,omitempty"`
	ReleaseStatusId   int        `json:"release_status_id"`
	EarliestEventDate *time.Time `json:"earliest_event_date,omitempty"`
	DaysUntilRelease  *int       `json:"days_until_release"`
	DaysUntilEvent    *int       `json:"days_until_event"`
}

func (h *ApiHandler) AdminGetUserEventNotification(gc *gin.Context) {
	db := h.DbPool
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	sql := app.Singleton.SqlAdminGetUserEventNotification

	rows, err := db.Query(ctx, sql, userId, 14, 30)
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
			&e.ReleaseStatusId,
			&e.ReleaseDate,
			&e.DaysUntilRelease,
			&e.OrganizerId,
			&e.OrganizerName,
			&e.EarliestEventDate,
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
