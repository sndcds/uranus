package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminGetSocialPublication(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-social-publication")
	if !app.ValidUUID(h.userUuid(gc)) {
		apiRequest.Error(http.StatusUnauthorized, "authentication required")
		return
	}
	publicationUuid, ok := socialPostUUID(gc, apiRequest)
	if !ok {
		return
	}
	ctx := gc.Request.Context()
	var result model.SocialPublication
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if txErr := h.socialPublicationPermission(gc, tx, publicationUuid); txErr != nil {
			return txErr
		}
		var err error
		result, err = scanSocialPublication(tx.QueryRow(ctx, h.socialPublicationQuery()+" WHERE p.uuid = $1", publicationUuid))
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
