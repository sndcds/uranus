package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetSocialPost(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-social-post")
	ctx := gc.Request.Context()
	postUuid, ok := socialPostUUID(gc, apiRequest)
	if !ok {
		return
	}

	var result model.SocialPost
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if _, txErr := h.socialPostPermission(gc, tx, postUuid); txErr != nil {
			return txErr
		}
		var err error
		result, err = scanSocialPost(tx.QueryRow(ctx, h.socialPostQuery()+" WHERE p.uuid = $1", postUuid))
		if err != nil {
			return socialPostDBError(err)
		}
		return nil
	})
	if txErr != nil {
		socialPostRespondError(apiRequest, txErr)
		return
	}
	apiRequest.Success(http.StatusOK, result)
}
