package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminDeleteSocialPost(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-social-post")
	ctx := gc.Request.Context()
	postUuid, ok := socialPostUUID(gc, apiRequest)
	if !ok {
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if _, txErr := h.socialPostPermission(gc, tx, postUuid); txErr != nil {
			return txErr
		}
		if txErr := h.socialPostPublishingConflict(gc, tx, postUuid); txErr != nil {
			return txErr
		}
		_, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s.social_post WHERE uuid = $1", h.DbSchema), postUuid)
		if err != nil {
			return socialPostDBError(err)
		}
		return nil
	})
	if txErr != nil {
		socialPostRespondError(apiRequest, txErr)
		return
	}
	apiRequest.SuccessNoData(http.StatusOK, "social post deleted")
}
