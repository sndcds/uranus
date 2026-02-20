package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminInitialEvent(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-initial-venue")

	type Payload struct {
		OrganizationId int    `json:"organization_id" binding:"required"`
		EventTitle     string `json:"event_title" binding:"required"`
	}
	payload, ok := grains_api.DecodeJSONBody[Payload](gc, apiRequest)
	if !ok {
		return
	}

	eventTitle := strings.TrimSpace(payload.EventTitle)
	if eventTitle == "" {
		apiRequest.Error(http.StatusBadRequest, "event_title cannot be empty")
		return
	}

	apiRequest.Metadata["organization_id"] = payload.OrganizationId
	apiRequest.Metadata["event_title"] = eventTitle

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		txErr := h.CheckOrganizationAllPermissions(gc, tx, userId, payload.OrganizationId, app.PermAddEvent)
		if txErr != nil {
			return txErr
		}

		newEventId := -1
		query := fmt.Sprintf(
			`INSERT INTO %s.event (organization_id, title, created_by) VALUES ($1, $2, $3) RETURNING id`,
			h.DbSchema)
		err := tx.QueryRow(ctx, query, payload.OrganizationId, eventTitle, userId).Scan(&newEventId)
		if err != nil {
			debugf(err.Error())
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Internal server error"),
			}
		}
		apiRequest.Metadata["event_id"] = newEventId
		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "event successfully created")
}
