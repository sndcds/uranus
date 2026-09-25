package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
	"github.com/sndcds/uranus/service"
)

type socialPublishWork struct {
	account  service.SocialPublishingAccount
	rendered model.RenderedPost
	err      error
	claimed  bool
}

// Claim the request's complete eligible set before any network call. The post
// lock serializes this short transaction with other claims and CRUD. No lock or
// transaction survives into publishSocialTarget.
func (h *ApiHandler) claimSocialPublish(gc *gin.Context, postUUID string, result *model.SocialPostPublish) ([]socialPublishWork, *ApiTxError) {
	var work []socialPublishWork
	ctx := gc.Request.Context()
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if _, err := h.socialPostPermission(gc, tx, postUUID); err != nil {
			return err
		}
		post, err := scanSocialPost(tx.QueryRow(ctx, h.socialPostQuery()+" WHERE p.uuid = $1", postUUID))
		if err != nil {
			return socialPostDBError(err)
		}
		if len(post.Targets) == 0 {
			return &ApiTxError{Code: http.StatusBadRequest, Message: "social post has no targets"}
		}
		busy := false
		for _, target := range post.Targets {
			if target.Status == "publishing" {
				busy = true
			}
		}
		var item *model.ContentItem
		for _, target := range post.Targets {
			if !busy && (target.Status == "draft" || target.Status == "failed") {
				var txErr *ApiTxError
				item, txErr = h.socialRenderContent(gc, tx, post)
				if txErr != nil {
					return txErr
				}
				break
			}
		}
		for _, target := range post.Targets {
			response := model.SocialTargetPublish{TargetUuid: target.Uuid, SocialAccountUuid: target.SocialAccountUuid, Status: target.Status, RemotePostID: target.RemotePostID}
			var account model.SocialAccount
			// Hold account metadata stable only until this snapshot/claim commits.
			err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT uuid, org_uuid, platform, remote_account_id, base_url, enabled
    FROM %s.social_account WHERE uuid = $1 FOR SHARE`, h.DbSchema), target.SocialAccountUuid).Scan(
				&account.Uuid, &account.OrgUuid, &account.Platform, &account.RemoteAccountID, &account.BaseURL, &account.Enabled)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return socialPostDBError(err)
			}
			response.Platform = account.Platform
			task := socialPublishWork{}
			if busy || (target.Status != "draft" && target.Status != "failed") {
				message := "target cannot be published from status " + target.Status
				if busy {
					message = "social post has an unresolved publishing target"
				}
				response.Error = &message
			} else {
				task.rendered, task.err = renderSocialTarget(account, *item)
				if errors.Is(err, pgx.ErrNoRows) {
					task.err = fmt.Errorf("social account not found")
				}
				if task.err == nil {
					// Fetch only the access token, into a redacting internal type.
					var secret socialAccountSecret
					if err := tx.QueryRow(ctx, fmt.Sprintf("SELECT access_token FROM %s.social_account WHERE uuid = $1", h.DbSchema), account.Uuid).Scan(&secret.Value); err != nil {
						return socialPostDBError(err)
					}
					accessToken := ""
					if secret.Value != nil {
						accessToken = *secret.Value
					}
					task.account = service.NewSocialPublishingAccount(account, accessToken)
				}
				command, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s.social_post_target
    SET status = 'publishing', published_at = NULL, remote_post_id = NULL, error = NULL, updated_at = CURRENT_TIMESTAMP
    WHERE uuid = $1 AND status IN ('draft', 'failed')`, h.DbSchema), target.Uuid)
				if err != nil {
					return socialPostDBError(err)
				}
				if command.RowsAffected() != 1 {
					return &ApiTxError{Code: http.StatusConflict, Message: "target is already being published"}
				}
				task.claimed = true
				response.Status, response.RemotePostID = "publishing", nil
			}
			result.Results = append(result.Results, response)
			work = append(work, task)
		}
		return nil
	})
	return work, txErr
}

func (h *ApiHandler) publishSocialTarget(ctx context.Context, task socialPublishWork) (model.SocialPublishResult, error) {
	if task.err != nil {
		return model.SocialPublishResult{}, task.err
	}
	publisher, err := service.NewSocialPublisher(task.account.Account.Platform, h.SocialHTTPClient, app.UranusInstance.Config.BaseApiUrl)
	if err != nil {
		return model.SocialPublishResult{}, err
	}
	return publisher.Publish(ctx, task.account, task.rendered)
}

func (h *ApiHandler) finishSocialPublish(ctx context.Context, result model.SocialTargetPublish) bool {
	// Remote calls remain cancellable. Only this bounded local bookkeeping
	// survives a disconnected caller so an observed success is still recorded.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	command, err := h.DbPool.Exec(ctx, fmt.Sprintf(`UPDATE %s.social_post_target
    SET status = $2, published_at = CASE WHEN $2 = 'published' THEN CURRENT_TIMESTAMP ELSE NULL END,
        remote_post_id = $3, error = $4, updated_at = CURRENT_TIMESTAMP
    WHERE uuid = $1 AND status = 'publishing'`, h.DbSchema), result.TargetUuid, result.Status, result.RemotePostID, result.Error)
	return err == nil && command.RowsAffected() == 1
}

func (h *ApiHandler) socialPostPublishingConflict(gc *gin.Context, tx pgx.Tx, postUUID string) *ApiTxError {
	var busy bool
	err := tx.QueryRow(gc.Request.Context(), fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s.social_post_target WHERE social_post_uuid = $1 AND status = 'publishing')", h.DbSchema), postUUID).Scan(&busy)
	if err != nil {
		return socialPostDBError(err)
	}
	if busy {
		return &ApiTxError{Code: http.StatusConflict, Message: "social post has an unresolved publishing target"}
	}
	return nil
}
