package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminToggleFavoriteEventDate(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-add-event-date-to-favorite-list")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	type Payload struct {
		OrgUuid          string  `json:"org_uuid" binding:"required"`
		FavoriteListUuid string  `json:"favorite_list_uuid" binding:"required"`
		EventUuid        string  `json:"event_uuid" binding:"required"`
		EventDateUuid    *string `json:"event_date_uuid"`
	}
	payload, ok := grains_api.DecodeJSONBody[Payload](gc, apiRequest)
	if !ok {
		apiRequest.PayloadError()
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		txErr := h.CheckAllOrgPermissionsTx(
			gc,
			tx,
			userUuid,
			payload.OrgUuid,
			app.UserPermAddToFavoriteList,
		)
		if txErr != nil {
			debugf(txErr.Error())
			return txErr
		}

		// First try deleting existing favorite
		deleteQuery := fmt.Sprintf(`
		DELETE FROM %s.favorite
		WHERE list_uuid = $1::uuid
		  AND context = $2
		  AND context_uuid = $3::uuid
	`, h.DbSchema)

		deleteResult, err := tx.Exec(
			ctx,
			deleteQuery,
			payload.FavoriteListUuid,
			"event-date",
			payload.EventDateUuid,
		)
		if err != nil {
			return TxInternalError(err)
		}

		// If something was deleted, we're done (toggle off)
		if deleteResult.RowsAffected() > 0 {
			apiRequest.Metadata["favorite_status"] = false
			return nil
		}

		// Otherwise insert favorite (toggle on)
		insertQuery := fmt.Sprintf(`
			INSERT INTO %s.favorite
			(list_uuid, created_by, context, context_uuid)
			VALUES ($1::uuid, $2::uuid, $3, $4::uuid)
			`, h.DbSchema)

		_, err = tx.Exec(
			ctx,
			insertQuery,
			payload.FavoriteListUuid,
			userUuid,
			"event-date",
			payload.EventDateUuid,
		)
		if err != nil {
			return TxInternalError(err)
		}

		apiRequest.Metadata["favorite_status"] = true

		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusCreated, "event date added successfully to favorite list")
}
