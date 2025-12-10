package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

type UpdateParticipationInfoRequest struct {
	ParticipationInfo    *string  `json:"participation_info"`
	MeetingPoint         *string  `json:"meeting_point"`
	MinAge               *int     `json:"min_age"`
	MaxAge               *int     `json:"max_age"`
	MaxAttendees         *int     `json:"max_attendees"`
	PriceTypeID          *int     `json:"price_type_id"`
	MinPrice             *float64 `json:"min_price"`
	MaxPrice             *float64 `json:"max_price"`
	CurrencyCode         *string  `json:"currency_code"`
	TicketAdvance        *bool    `json:"ticket_advance"`
	TicketRequired       *bool    `json:"ticket_required"`
	RegistrationRequired *bool    `json:"registration_required"`
	OccasionTypeID       *int     `json:"occasion_type_id"`
}

func (h *ApiHandler) AdminUpdateEventParticipationInfos(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	var req UpdateParticipationInfoRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build SQL
	setClauses := []string{}
	args := []interface{}{eventId}
	argIndex := 2

	if req.ParticipationInfo != nil {
		setClauses = append(setClauses, fmt.Sprintf("participation_info = $%d", argIndex))
		args = append(args, *req.ParticipationInfo)
		argIndex++
	}
	if req.MeetingPoint != nil {
		setClauses = append(setClauses, fmt.Sprintf("meeting_point = $%d", argIndex))
		args = append(args, *req.MeetingPoint)
		argIndex++
	}
	if req.MinAge != nil {
		setClauses = append(setClauses, fmt.Sprintf("min_age = $%d", argIndex))
		args = append(args, *req.MinAge)
		argIndex++
	}
	if req.MaxAge != nil {
		setClauses = append(setClauses, fmt.Sprintf("max_age = $%d", argIndex))
		args = append(args, *req.MaxAge)
		argIndex++
	}
	if req.MaxAttendees != nil {
		setClauses = append(setClauses, fmt.Sprintf("max_attendees = $%d", argIndex))
		args = append(args, *req.MaxAttendees)
		argIndex++
	}
	if req.PriceTypeID != nil {
		setClauses = append(setClauses, fmt.Sprintf("price_type_id = $%d", argIndex))
		args = append(args, *req.PriceTypeID)
		argIndex++
	}
	if req.MinPrice != nil {
		setClauses = append(setClauses, fmt.Sprintf("min_price = $%d", argIndex))
		args = append(args, *req.MinPrice)
		argIndex++
	}
	if req.MaxPrice != nil {
		setClauses = append(setClauses, fmt.Sprintf("max_price = $%d", argIndex))
		args = append(args, *req.MaxPrice)
		argIndex++
	}
	if req.CurrencyCode != nil {
		setClauses = append(setClauses, fmt.Sprintf("currency_code = $%d", argIndex))
		args = append(args, *req.CurrencyCode)
		argIndex++
	}
	if req.TicketAdvance != nil {
		setClauses = append(setClauses, fmt.Sprintf("ticket_advance = $%d", argIndex))
		args = append(args, *req.TicketAdvance)
		argIndex++
	}
	if req.TicketRequired != nil {
		setClauses = append(setClauses, fmt.Sprintf("ticket_required = $%d", argIndex))
		args = append(args, *req.TicketRequired)
		argIndex++
	}
	if req.RegistrationRequired != nil {
		setClauses = append(setClauses, fmt.Sprintf("registration_required = $%d", argIndex))
		args = append(args, *req.RegistrationRequired)
		argIndex++
	}
	if req.OccasionTypeID != nil {
		setClauses = append(setClauses, fmt.Sprintf("occasion_type_id = $%d", argIndex))
		args = append(args, *req.OccasionTypeID)
		argIndex++
	}

	if len(setClauses) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	sqlQuery := fmt.Sprintf(`
        UPDATE %s.event
        SET %s
        WHERE id = $1`,
		dbSchema,
		strings.Join(setClauses, ", "),
	)

	res, err := pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update event: %v", err)})
		return
	}

	if res.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventId,
		"updated":  setClauses,
		"message":  "event participation info updated",
	})
}
