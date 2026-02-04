package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/model"
)

type participationInfoReq struct {
	ParticipationInfo *string         `json:"participation_info"`
	MeetingPoint      *string         `json:"meeting_point"`
	MinAge            *int            `json:"min_age"`
	MaxAge            *int            `json:"max_age"`
	MaxAttendees      *int            `json:"max_attendees"`
	PriceType         model.PriceType `json:"price_type"`
	MinPrice          *float64        `json:"min_price"`
	MaxPrice          *float64        `json:"max_price"`
	Currency          *string         `json:"currency"`
	TicketFlags       []string        `json:"ticket_flags"`
	OccasionTypeID    *int            `json:"occasion_type_id"`
}

func (h *ApiHandler) AdminUpdateEventParticipationInfos(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	var req participationInfoReq
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build SQL
	setClauses := []string{}
	args := []interface{}{eventId}
	argIndex := 2

	setClauses = append(setClauses, fmt.Sprintf("participation_info = $%d", argIndex))
	args = append(args, req.ParticipationInfo)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("meeting_point = $%d", argIndex))
	args = append(args, req.MeetingPoint)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("min_age = $%d", argIndex))
	args = append(args, req.MinAge)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("max_age = $%d", argIndex))
	args = append(args, req.MaxAge)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("max_attendees = $%d", argIndex))
	args = append(args, req.MaxAttendees)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("price_type = $%d", argIndex))
	args = append(args, req.PriceType)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("min_price = $%d", argIndex))
	args = append(args, req.MinPrice)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("max_price = $%d", argIndex))
	args = append(args, req.MaxPrice)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("currency = $%d", argIndex))
	args = append(args, req.Currency)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("ticket_flags = $%d", argIndex))
	args = append(args, req.TicketFlags)
	argIndex++

	setClauses = append(setClauses, fmt.Sprintf("occasion_type_id = $%d", argIndex))
	args = append(args, req.OccasionTypeID)
	argIndex++

	if len(setClauses) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		query := fmt.Sprintf(`UPDATE %s.event SET %s WHERE id = $1`,
			h.DbSchema,
			strings.Join(setClauses, ", "),
		)

		fmt.Println("query:", query)

		res, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to update event: %v", err),
			}
		}

		if res.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  fmt.Errorf("event not found"),
			}
		}

		err = RefreshEventProjections(ctx, tx, "event", []int{eventId})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %v", err),
			}
		}

		return nil
	})
	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":  "event participation info updated",
		"event_id": eventId,
		"updated":  setClauses,
	})
}
