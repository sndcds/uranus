package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

/*
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
*/

type eventDateReq struct {
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
	userId := gc.GetInt("user-id")

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	dateId, ok := ParamInt(gc, "dateId")
	if !ok {
		dateId = -1 // New event date must be inserted
	}

	// TODO: Check Permissions!

	var req eventDateReq
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newEventDateId := -1

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Compute entry_time
		var entryTime interface{}
		if req.EntryTime != nil && *req.EntryTime != "" {
			entryTime = *req.EntryTime
		} else {
			entryTime = nil
		}

		if dateId < 0 { // Insert new event date
			query := fmt.Sprintf(`
INSERT INTO %s.event_date 
(event_id, venue_id, space_id, start_date, start_time, end_date, end_time, entry_time, all_day, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
				h.Config.DbSchema)
			err := tx.QueryRow(ctx, query,
				eventId,
				req.VenueId,
				req.SpaceId,
				req.StartDate,
				req.StartTime,
				req.EndDate,
				req.EndTime,
				entryTime,
				req.AllDay,
				userId,
			).Scan(&newEventDateId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to insert event date: %v", err),
				}
			}

			err = RefreshEventProjections(ctx, tx, "event_date", []int{newEventDateId})
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("refresh projection tables failed: %v", err),
				}
			}
		} else { // Update existing event date
			query := fmt.Sprintf(`
UPDATE %s.event_date
SET venue_id = $1, space_id = $2, start_date = $3, start_time = $4, end_date = $5, end_time = $6, entry_time = $7, all_day = $8
WHERE event_id = $9 AND id = $10`,
				h.Config.DbSchema)
			cmdTag, err := tx.Exec(ctx, query,
				req.VenueId,
				req.SpaceId,
				req.StartDate,
				req.StartTime,
				req.EndDate,
				req.EndTime,
				entryTime,
				req.AllDay,
				eventId,
				dateId,
			)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to update event date: %v", err),
				}
			}

			if cmdTag.RowsAffected() == 0 {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  fmt.Errorf("event date not found"),
				}
			}

			err = RefreshEventProjections(ctx, tx, "event_date", []int{dateId})
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("refresh projection tables failed: %v", err),
				}
			}
		}

		return nil
	})
	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	if newEventDateId >= 0 {
		gc.JSON(http.StatusOK, gin.H{
			"message":       "event date created successfully",
			"event_id":      eventId,
			"event_date_id": newEventDateId,
		})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":       "event date updated successfully",
		"event_id":      eventId,
		"event_date_id": dateId,
	})
}
