package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminDeleteSpace(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-space")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	err := h.VerifyUserPassword(gc, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	spaceUuid := gc.Param("spaceUuid")
	if spaceUuid == "" {
		apiRequest.Required("spaceUuid is required")
		return
	}
	apiRequest.SetMeta("space_uuid", spaceUuid)

	query := fmt.Sprintf(`DELETE FROM %s.space WHERE uuid = $1::uuid`, h.DbSchema)

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		// Permission check
		orgUuid, err := h.GetOrgUuidBySpaceUuidTx(gc, tx, spaceUuid)
		if err != nil {
			return TxInternalError(err)
		}

		txErr := h.CheckOrgPermissionTx(gc, tx, userUuid, orgUuid, app.UserPermDeleteSpace)
		if txErr != nil {
			return txErr
		}

		cmdTag, err := tx.Exec(ctx, query, spaceUuid)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Failed to delete space"),
			}
		}

		if cmdTag.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  fmt.Errorf("Space not found"),
			}
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Message)
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "Space deleted successfully")
}
