package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

type EventDateLocationInput struct {
	Name        *string  `json:"name"`
	Street      *string  `json:"street"`
	HouseNumber *string  `json:"house_number"`
	PostalCode  *string  `json:"postal_code"`
	City        *string  `json:"city"`
	CountryCode *string  `json:"country_code"`
	StateCode   *string  `json:"state_code"`
	Latitude    *float64 `json:"lat"`
	Longitude   *float64 `json:"lon"`
	Description *string  `json:"description"`
}

type EventDateInput struct {
	StartDate string  `json:"start_date" binding:"required"`
	StartTime string  `json:"start_time" binding:"required"`
	EndDate   *string `json:"end_date"`
	EndTime   *string `json:"end_time"`
	EntryTime *string `json:"entry_time"`
	AllDay    bool    `json:"all_day"`
	VenueId   *int    `json:"venue_id"`
	SpaceId   *int    `json:"space_id"`
}

func (h *ApiHandler) AdminUpsertEventDate(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	userId := gc.GetInt("user-id")

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	dateId, ok := ParamInt(gc, "dateId")
	fmt.Println("dateId:", dateId)
	if !ok {
		dateId = -1 // New event date must be inserted
	}
	fmt.Println("dateId 2:", dateId)

	fmt.Println("userId:", userId)
	fmt.Println("eventId:", eventId)

	// TODO: Check Permissions!

	var incoming EventDateInput
	if err := gc.ShouldBindJSON(&incoming); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Compute entry_time
	var entryTime interface{}
	if incoming.EntryTime != nil && *incoming.EntryTime != "" {
		entryTime = *incoming.EntryTime
	} else {
		entryTime = nil
	}

	if dateId < 0 {
		// Insert new event date
		insertSql := fmt.Sprintf(`
			INSERT INTO %s.event_date 
				(event_id, venue_id, space_id, start_date, start_time, end_date, end_time, entry_time, all_day, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id
		`, h.Config.DbSchema)

		var newEventDateId int
		err = tx.QueryRow(ctx, insertSql,
			eventId,
			incoming.VenueId,
			incoming.SpaceId,
			incoming.StartDate,
			incoming.StartTime,
			incoming.EndDate,
			incoming.EndTime,
			entryTime,
			incoming.AllDay,
			userId,
		).Scan(&newEventDateId)
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
			"event_date_id": newEventDateId,
			"message":       "event date created successfully",
		})
		return
	}

	// Update existing event date
	sqlUpdate := fmt.Sprintf(`
		UPDATE %s.event_date
		SET venue_id = $1, space_id = $2, start_date = $3, start_time = $4, end_date = $5, end_time = $6, entry_time = $7, all_day = $8
		WHERE event_id = $9 AND id = $10 
	`, h.Config.DbSchema)

	cmdTag, err := tx.Exec(ctx, sqlUpdate,
		incoming.VenueId,
		incoming.SpaceId,
		incoming.StartDate,
		incoming.StartTime,
		incoming.EndDate,
		incoming.EndTime,
		entryTime,
		incoming.AllDay,
		eventId,
		dateId,
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
		"event_date_id": dateId,
		"message":       "event date updated successfully",
	})
}
