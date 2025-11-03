package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type EventNotification struct {
	ID            int        `json:"id"`
	Title         string     `json:"title"`
	ReleaseDate   *time.Time `json:"release_date,omitempty"`
	ReleaseStatus int        `json:"release_status_id"`
	OrganizerID   int        `json:"organizer_id"`
}

func (h *ApiHandler) AdminGetUserEventNotification(gc *gin.Context) {
	db := h.DbPool
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	sql := app.Singleton.SqlAdminGetUserEventNotification

	rows, err := db.Query(ctx, sql, userId)
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
			&e.ID,
			&e.Title,
			&e.ReleaseDate,
			&e.ReleaseStatus,
			&e.OrganizerID,
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
