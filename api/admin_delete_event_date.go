package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminDeleteEventDate(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-event-date")
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

	eventDateUuid := gc.Param("dateUuid")
	if eventDateUuid == "" {
		apiRequest.Required("dateUuid is required")
		return
	}
	apiRequest.SetMeta("event_date_uuid", eventDateUuid)

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
			`DELETE FROM %s.event_date WHERE uuid = $1::uuid AND event_uuid = $2::uuid`,
			h.DbSchema,
		)
		cmdTag, err := tx.Exec(ctx, query, eventDateUuid, eventUuid)
		if err != nil {
			return TxInternalError(err)
		}

		if cmdTag.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  fmt.Errorf("event date not found"),
			}
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Message)
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "event date deleted successfully")
}
