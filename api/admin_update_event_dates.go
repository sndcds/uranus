package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Eventually to be removed
func (h *ApiHandler) AdminUpdateEventDates(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	var req struct {
		Dates []struct {
			StartDate string  `json:"start_date" binding:"required"`
			StartTime string  `json:"start_time" binding:"required"`
			EndDate   *string `json:"end_date"`
			EndTime   *string `json:"end_time"`
			EntryTime *string `json:"entry_time"`
			AllDay    bool    `json:"all_day"`
			VenueId   *int    `json:"venue_id"`
			SpaceId   *int    `json:"space_id"`
		} `json:"dates" binding:"required,dive"`
	}
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sqlDelete := fmt.Sprintf(`DELETE FROM %s.event_date WHERE event_id = $1`, dbSchema)
	if _, err = tx.Exec(ctx, sqlDelete, eventId); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete existing event dates: %v", err)})
		return
	}

	sqlInsert := fmt.Sprintf(
		`INSERT INTO %s.event_date 
        (event_id, venue_id, space_id, start, "end", entry_time, all_day)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		dbSchema,
	)

	for _, d := range req.Dates {
		startTimestamp := d.StartDate + " " + d.StartTime

		// Compute endTimestamp safely
		var endTimestamp *string
		if d.EndTime != nil && *d.EndTime != "" {
			var endDate string
			if d.EndDate != nil && *d.EndDate != "" {
				endDate = *d.EndDate
			} else {
				endDate = d.StartDate
			}
			t := endDate + " " + *d.EndTime
			endTimestamp = &t
		}

		// Compute entry_time
		var entryTime interface{}
		if d.EntryTime != nil && *d.EntryTime != "" {
			entryTime = *d.EntryTime
		} else {
			entryTime = nil
		}

		fmt.Println("eventId", eventId)
		fmt.Println("d.VenueId", d.VenueId)
		fmt.Println("d.SpaceId", d.SpaceId)
		fmt.Println("startTimestamp", startTimestamp)
		fmt.Println("endTimestamp", endTimestamp)
		fmt.Println("entryTime", entryTime)
		fmt.Println("d.AllDay", d.AllDay)

		_, err = tx.Exec(ctx, sqlInsert,
			eventId,
			d.VenueId,
			d.SpaceId,
			startTimestamp,
			endTimestamp,
			entryTime,
			d.AllDay,
		)
		if err != nil {
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
