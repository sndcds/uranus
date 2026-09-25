package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminCreateSocialPost(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-create-social-post")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	fmt.Println("userUuid", userUuid)
	payload := decodeSocialPost(gc, apiRequest, true)
	if payload == nil {
		return
	}
	postUuid, err := grains_uuid.Uuidv7String()
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	fmt.Println("postUuid", postUuid)

	var result model.SocialPost
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if txErr := h.CheckAllOrgPermissionsTx(gc, tx, userUuid, *payload.OrgUuid.Value, app.UserPermEditOrg); txErr != nil {
			return txErr
		}
		query := fmt.Sprintf(`
			INSERT INTO %s.social_post
			(uuid, org_uuid, source_type, source_uuid, created_by)
			VALUES ($1, $2, $3, $4, $5)`,
			h.DbSchema)
		fmt.Println("query", query)

		_, err := tx.Exec(ctx, query, postUuid, payload.OrgUuid.Value, payload.SourceType.Value, payload.SourceUuid.Value, userUuid)
		if err != nil {
			return socialPostDBError(err)
		}
		if payload.Targets.Set {
			if txErr := h.updateSocialPostTargets(gc, tx, postUuid, *payload.OrgUuid.Value, *payload.Targets.Value); txErr != nil {
				return txErr
			}
		}
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

	apiRequest.Success(http.StatusCreated, result)
}
