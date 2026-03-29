package api

import (
	"errors"
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
	useUuid := h.userUuid(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-initial-venue")

	type Payload struct {
		OrgUuid    string `json:"org_uuid" binding:"required"`
		EventTitle string `json:"event_title" binding:"required"`
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

	apiRequest.Metadata["org_uuid"] = payload.OrgUuid
	apiRequest.Metadata["event_title"] = eventTitle

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		txErr := h.CheckOrganizationAllPermissions(gc, tx, useUuid, payload.OrgUuid, app.PermAddEvent)
		if txErr != nil {
			return txErr
		}

		newEventId := -1
		query := fmt.Sprintf(
			`INSERT INTO %s.event (organization_id, title, created_by) VALUES ($1, $2, $3) RETURNING id`,
			h.DbSchema)
		err := tx.QueryRow(ctx, query, payload.OrgUuid, eventTitle, useUuid).Scan(&newEventId)
		if err != nil {
			debugf(err.Error())
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("Internal server error"),
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
