package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminUpsertEventDate(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
	}

	var req struct {
		EventDateId int     `json:"event_date_id"` // -1 for new
		StartDate   string  `json:"start_date" binding:"required"`
		StartTime   string  `json:"start_time" binding:"required"`
		EndDate     *string `json:"end_date"`
		EndTime     *string `json:"end_time"`
		EntryTime   *string `json:"entry_time"`
		AllDay      bool    `json:"all_day"`
		VenueId     *int    `json:"venue_id"`
		SpaceId     *int    `json:"space_id"`
		/*
			VisitorInfoFlags int64   `json:"visitor_info_flags" binding:"required"` TODO: Implement
			ticket_link text,
			duration integer,
			custom text,
			status text CHECK (status = ANY (ARRAY['planned'::text, 'confirmed'::text, 'rescheduled'::text, 'cancelled'::text, 'draft'::text])),
			availability_status text,
			all_day boolean,
			venue_id integer REFERENCES uranus.venue(id) ON DELETE SET NULL,
			accessibility_info text
		*/
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

	startTimestamp := req.StartDate + " " + req.StartTime

	// Compute endTimestamp safely
	var endTimestamp interface{}
	if req.EndTime != nil && *req.EndTime != "" {
		endDate := req.StartDate
		if req.EndDate != nil && *req.EndDate != "" {
			endDate = *req.EndDate
		}
		t := endDate + " " + *req.EndTime
		endTimestamp = t
	} else {
		endTimestamp = nil
	}

	// Compute entry_time
	var entryTime interface{}
	if req.EntryTime != nil && *req.EntryTime != "" {
		entryTime = *req.EntryTime
	} else {
		entryTime = nil
	}

	if req.EventDateId < 0 {
		// Insert new event date
		sqlInsert := fmt.Sprintf(`
			INSERT INTO %s.event_date 
				(event_id, venue_id, space_id, start, "end", entry_time, all_day)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`, dbSchema)

		var newId int
		err = tx.QueryRow(ctx, sqlInsert,
			eventId,
			req.VenueId,
			req.SpaceId,
			startTimestamp,
			endTimestamp,
			entryTime,
			req.AllDay,
		).Scan(&newId)
		if err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event date: %v", err)})
			return
		}

		if err = tx.Commit(ctx); err != nil {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
			return
		}

		gc.JSON(http.StatusOK, gin.H{
			"event_id":      eventId,
			"event_date_id": newId,
			"message":       "event date created successfully",
		})
		return
	}

	// Update existing event date
	sqlUpdate := fmt.Sprintf(`
		UPDATE %s.event_date
		SET venue_id = $1, space_id = $2, start = $3, "end" = $4, entry_time = $5, all_day = $6
		WHERE id = $7 AND event_id = $8
	`, dbSchema)

	cmdTag, err := tx.Exec(ctx, sqlUpdate,
		req.VenueId,
		req.SpaceId,
		startTimestamp,
		endTimestamp,
		entryTime,
		req.AllDay,
		req.EventDateId,
		eventId,
	)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update event date: %v", err)})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event date not found"})
		return
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id":      eventId,
		"event_date_id": req.EventDateId,
		"message":       "event date updated successfully",
	})
}
