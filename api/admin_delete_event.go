package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminDeleteEvent(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-event")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	err := h.VerifyUserPassword(gc, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Required("eventUuid is required")
		return
	}
	apiRequest.SetMeta("event_uuid", eventUuid)

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// Permission check
		orgUuid, err := h.GetOrgUuidByEventUuidTx(gc, tx, eventUuid)
		if err != nil {
			return TxInternalError(err)
		}

		txErr := h.CheckOrgPermissionTx(gc, tx, userUuid, orgUuid, app.UserPermDeleteEvent)
		if txErr != nil {
			return txErr
		}

		query := fmt.Sprintf(
			`DELETE FROM %s.event WHERE uuid = $1::uuid`,
			h.DbSchema,
		)

		cmdTag, err := tx.Exec(ctx, query, eventUuid)
		if err != nil {
			return TxInternalError(err)
		}

		if cmdTag.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  fmt.Errorf("event not found"),
			}
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Message)
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "event deleted successfully")
}
