package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminCheckFavoriteListEventDate(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-check-favorite-list-event-date")
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

	debugf("OrgUuid: %s", payload.OrgUuid)
	debugf("FavoriteListUuid: %s", payload.FavoriteListUuid)
	debugf("EventUuid: %s", payload.EventUuid)
	if payload.EventDateUuid != nil {
		debugf("EventDateUuid: %s", *payload.EventDateUuid)
	}

	var favoriteStatus bool

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

		query := fmt.Sprintf(`
			SELECT EXISTS (
				SELECT 1
				FROM %s.favorite
				WHERE list_uuid = $1::uuid
				  AND context = $2
				  AND context_uuid = $3::uuid
			)
		`, h.DbSchema)

		err := tx.QueryRow(
			ctx,
			query,
			payload.FavoriteListUuid,
			"event-date",
			payload.EventDateUuid,
		).Scan(&favoriteStatus)

		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  err,
			}
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.Metadata["favorite_status"] = favoriteStatus
	apiRequest.SuccessNoData(http.StatusOK, "favorite status successfully loaded")
}
