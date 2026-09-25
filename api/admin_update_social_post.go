package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminUpdateSocialPost(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-social-post")
	ctx := gc.Request.Context()
	postUuid, ok := socialPostUUID(gc, apiRequest)
	if !ok {
		return
	}
	payload := decodeSocialPost(gc, apiRequest, false)
	if payload == nil {
		return
	}

	setClauses := []string{"updated_at = CURRENT_TIMESTAMP"}
	var args []interface{}
	argPos := 1
	argPos = addUpdateClauseNullable("source_type", payload.SourceType, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("source_uuid", payload.SourceUuid, &setClauses, &args, argPos)
	args = append(args, postUuid)
	query := fmt.Sprintf("UPDATE %s.social_post SET %s WHERE uuid = $%d",
		h.DbSchema, strings.Join(setClauses, ", "), argPos)

	var result model.SocialPost
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		orgUuid, txErr := h.socialPostPermission(gc, tx, postUuid)
		if txErr != nil {
			return txErr
		}
		if payload.OrgUuid.Set && *payload.OrgUuid.Value != orgUuid {
			return &ApiTxError{Code: http.StatusBadRequest, Message: "org_uuid cannot be changed"}
		}
		if payload.Targets.Set {
			if txErr := h.updateSocialPostTargets(gc, tx, postUuid, orgUuid, *payload.Targets.Value); txErr != nil {
				return txErr
			}
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return socialPostDBError(err)
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
