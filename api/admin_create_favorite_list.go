package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminCreateFavoriteList(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-create-favorite-list")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	type Payload struct {
		OrgUuid          string `json:"org_uuid" binding:"required"`
		FavoriteListName string `json:"favorite_list_name" binding:"required"`
	}
	payload, ok := grains_api.DecodeJSONBody[Payload](gc, apiRequest)
	if !ok {
		apiRequest.PayloadError()
		return
	}

	favoriteListName := strings.TrimSpace(payload.FavoriteListName)
	if favoriteListName == "" {
		apiRequest.Error(http.StatusBadRequest, "favorite_list_name cannot be empty")
		return
	}

	apiRequest.Metadata["org_uuid"] = payload.OrgUuid
	apiRequest.Metadata["portal_name"] = favoriteListName

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		txErr := h.CheckAllOrgPermissionsTx(gc, tx, userUuid, payload.OrgUuid, app.UserPermAddFavoriteList)
		if txErr != nil {
			debugf(txErr.Error())
			return txErr
		}

		favoriteListUuid, err := grains_uuid.Uuidv7String()
		apiRequest.Metadata["favorite_list_uuid"] = favoriteListUuid
		query := fmt.Sprintf(`
			INSERT INTO %s.favorite_list (uuid, org_uuid, created_by, name)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4)`,
			h.DbSchema)
		_, err = tx.Exec(ctx, query, favoriteListUuid, payload.OrgUuid, userUuid, favoriteListName)
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

	apiRequest.SuccessNoData(http.StatusCreated, "favorite list successfully created")
}
