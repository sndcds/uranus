package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
	"github.com/sndcds/uranus/service"
)

func (h *ApiHandler) AdminReconcileSocialPublication(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-reconcile-social-publication")
	if !app.ValidUUID(h.userUuid(gc)) {
		apiRequest.Error(http.StatusUnauthorized, "authentication required")
		return
	}
	publicationUuid, ok := socialPostUUID(gc, apiRequest)
	if !ok {
		return
	}
	var payload *struct {
		Outcome         string                `json:"outcome"`
		RemotePostID    NullableField[string] `json:"remote_post_id"`
		ConfirmInactive bool                  `json:"confirm_inactive"`
	}
	if err := grains_api.BindJSONStrict(gc, &payload); err != nil || payload == nil {
		apiRequest.PayloadError()
		return
	}
	if payload.Outcome != "published" && payload.Outcome != "failed" {
		apiRequest.Error(http.StatusBadRequest, "outcome must be published or failed")
		return
	}
	if payload.Outcome == "failed" && payload.RemotePostID.Set {
		apiRequest.Error(http.StatusBadRequest, "remote_post_id is not allowed for failed outcome")
		return
	}

	ctx := gc.Request.Context()
	var result model.SocialPublication
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if txErr := h.socialPublicationPermission(gc, tx, publicationUuid); txErr != nil {
			return txErr
		}
		publication, err := scanSocialPublication(tx.QueryRow(ctx,
			h.socialPublicationQuery()+" WHERE p.uuid = $1 FOR UPDATE", publicationUuid))
		if err != nil {
			return socialPostDBError(err)
		}
		if publication.Status != "publishing" && publication.Status != "uncertain" {
			return &ApiTxError{Code: http.StatusConflict, Message: "publication is already resolved"}
		}
		// A durable claim alone cannot distinguish a crashed publisher from a
		// live one. The operator must stop publishers before recovering this case.
		if publication.Status == "publishing" && !payload.ConfirmInactive {
			return &ApiTxError{Code: http.StatusConflict, Message: "stop publishers and confirm_inactive before reconciling a publishing attempt"}
		}
		if payload.Outcome == "published" && (payload.RemotePostID.Value == nil ||
			!service.ValidSocialRemotePostID(publication.Platform, *payload.RemotePostID.Value)) {
			return &ApiTxError{Code: http.StatusBadRequest, Message: "valid remote_post_id is required"}
		}
		var message *string
		if payload.Outcome == "failed" {
			value := "operator confirmed no remote post was created"
			message = &value
		}
		_, err = tx.Exec(ctx, fmt.Sprintf(`UPDATE %s.social_publication
			SET status = $2, remote_post_id = $3, error = $4, finished_at = CURRENT_TIMESTAMP,
				reconciled_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE uuid = $1`, h.DbSchema), publicationUuid, payload.Outcome, payload.RemotePostID.Value, message)
		if err != nil {
			return socialPostDBError(err)
		}
		command, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s.social_post_target
			SET status = $2, remote_post_id = $3, error = $4,
				published_at = CASE WHEN $2 = 'published' THEN CURRENT_TIMESTAMP ELSE NULL END,
				updated_at = CURRENT_TIMESTAMP
			WHERE uuid = $1 AND status = 'publishing'`, h.DbSchema),
			publication.SocialPostTargetUuid, payload.Outcome, payload.RemotePostID.Value, message)
		if err != nil {
			return socialPostDBError(err)
		}
		if command.RowsAffected() != 1 {
			return &ApiTxError{Code: http.StatusConflict, Message: "target is no longer publishing"}
		}
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
