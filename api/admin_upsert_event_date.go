package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminUpsertEventDate(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	fmt.Println("userId", userId)

	eventId, ok := ParamInt(gc, "eventId")
	fmt.Println("eventId", eventId)
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
		return
	}

	var req model.EventDatePayload
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert the struct to JSON for debugging/logging
	reqJSON, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		fmt.Println("Failed to marshal req:", err)
	} else {
		fmt.Println(string(reqJSON))
	}

	eventDateId := ParamIntDefault(gc, "dateId", -1)
	newEventDateId := -1

	fmt.Println("eventDateId", eventDateId)

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		if eventDateId < 0 {
			// Insert
			// Check permissions, we need an 'organizationId' first
			organizationId, err := h.GetOrganizationIdByEvenId(gc, tx, eventId)
			if err != nil {
				return ApiErrInternal("%v", err)
			}
			if organizationId < 0 {
				return ApiErrInternal("internal organizationId failed")
			}

			permissions, err := h.GetUserOrganizationPermissions(gc, tx, userId, organizationId)
			if err != nil {
				return ApiErrInternal("%v", err)
			}
			if !permissions.HasAny(app.PermAddEvent | app.PermEditEvent) {
				return ApiErrForbidden("")
			}

			query := fmt.Sprintf(`
INSERT INTO %s.event_date 
(event_id, venue_id, space_id, start_date, start_time, end_date, end_time, entry_time, all_day, ticket_link, availability_status_id, accessibility_info, custom, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
RETURNING id`, h.DbSchema)

			err = tx.QueryRow(ctx, query,
				eventId,
				req.VenueId,
				req.SpaceId,
				req.StartDate,
				req.StartTime,
				req.EndDate,
				req.EndTime,
				req.EntryTime,
				req.AllDay,
				req.TicketLink,
				req.AvailabilityStatusId,
				req.AccessibilityInfo,
				req.Custom,
				userId,
			).Scan(&newEventDateId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to insert event date: %v", err),
				}
			}
		} else {
			// Update
			fmt.Println("Update event date")

			query := fmt.Sprintf(`
UPDATE %s.event_date
SET venue_id = $1,
    space_id = $2,
    start_date = $3,
    start_time = $4,
    end_date = $5,
    end_time = $6,
    entry_time = $7,
    all_day = $8,
    ticket_link = $9,
    availability_status_id = $10,
    accessibility_info = $11,
    custom = $12
WHERE event_id = $13 AND id = $14
RETURNING id`, h.DbSchema)

			err := tx.QueryRow(ctx, query,
				req.VenueId,
				req.SpaceId,
				req.StartDate,
				req.StartTime,
				req.EndDate,
				req.EndTime,
				req.EntryTime,
				req.AllDay,
				req.TicketLink,
				req.AvailabilityStatusId,
				req.AccessibilityInfo,
				req.Custom,
				eventId,
				eventDateId,
			).Scan(&newEventDateId)
			if err != nil {
				if err == pgx.ErrNoRows {
					return &ApiTxError{
						Code: http.StatusNotFound,
						Err:  fmt.Errorf("event date not found"),
					}
				}
				return ApiErrInternal("%v", err)
			}
		}
		fmt.Println("newEventDateId", newEventDateId)

		// Refresh projections
		if err := RefreshEventProjections(ctx, tx, "event_date", []int{newEventDateId}); err != nil {
			return ApiErrInternal("refresh projection tables failed: %v", err)
		}

		return nil
	})

	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	action := "updated"
	if eventDateId < 0 {
		action = "created"
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("event date %s successfully", action),
		"event_id":      eventId,
		"event_date_id": newEventDateId,
	})
}
