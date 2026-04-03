package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminInitialEvent(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-initial-venue")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	type Payload struct {
		OrgUuid    string `json:"org_uuid" binding:"required"`
		EventTitle string `json:"event_title" binding:"required"`
	}
	payload, ok := grains_api.DecodeJSONBody[Payload](gc, apiRequest)
	if !ok {
		apiRequest.PayloadError()
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
		txErr := h.CheckOrganizationAllPermissions(gc, tx, userUuid, payload.OrgUuid, app.PermAddEvent)
		if txErr != nil {
			debugf(".... 1")
			return txErr
		}

		query := fmt.Sprintf(
			`INSERT INTO %s.event (uuid, org_uuid, title, created_by) VALUES ($1::uuid, $2::uuid, $3, $4::uuid)`,
			h.DbSchema)
		eventUuid, err := grains_uuid.Uuidv7String()
		_, err = tx.Exec(ctx, query, eventUuid, payload.OrgUuid, eventTitle, userUuid)
		if err != nil {
			debugf(".... 2")
			debugf(err.Error())
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  errors.New("Internal server error"),
			}
		}
		apiRequest.Metadata["event_uuid"] = eventUuid
		return nil
	})
	if txErr != nil {
		debugf("1 ... %s", txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "event successfully created")
}
