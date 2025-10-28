package api_admin

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type UpdateEventDatesRequest struct {
	Dates []struct {
		StartDate string  `json:"start_date" binding:"required"`
		StartTime string  `json:"start_time" binding:"required"`
		EndDate   *string `json:"end_date"`
		EndTime   *string `json:"end_time"`
		EntryTime *string `json:"entry_time"`
		AllDay    bool    `json:"all_day"`
		SpaceId   *int    `json:"space_id"`
	} `json:"dates" binding:"required,dive"`
}

func UpdateEventDatesHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool
	dbSchema := app.Singleton.Config.DbSchema

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	var req UpdateEventDatesRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Optional: delete all existing event dates for this event
	sqlDelete := fmt.Sprintf(`DELETE FROM %s.event_date WHERE event_id = $1`, dbSchema)
	if _, err = tx.Exec(ctx, sqlDelete, eventId); err != nil {
		_ = tx.Rollback(ctx)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete existing event dates: %v", err)})
		return
	}

	sqlInsert := fmt.Sprintf(
		`INSERT INTO %s.event_date 
        (event_id, space_id, start, "end", entry_time, all_day)
        VALUES ($1, $2, $3, $4, $5, $6)`,
		dbSchema,
	)

	for _, d := range req.Dates {
		startTimestamp := d.StartDate + " " + d.StartTime
		var endTimestamp *string
		if d.EndDate != nil && d.EndTime != nil {
			t := *d.EndDate + " " + *d.EndTime
			endTimestamp = &t
		}

		_, err = tx.Exec(ctx, sqlInsert,
			eventId,
			d.SpaceId,
			startTimestamp,
			endTimestamp,
			d.EntryTime,
			d.AllDay,
		)
		if err != nil {
			_ = tx.Rollback(ctx)
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event date: %v", err)})
			return
		}
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventId,
		"message":  "event dates updated successfully",
	})
}
