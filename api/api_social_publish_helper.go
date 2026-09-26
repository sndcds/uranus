package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_uuid"
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

// The worker holds the post lock. Manual and scheduled requests share this
// render, fingerprint, credential and publication claim path.
func (h *ApiHandler) prepareSocialPublishTx(
	ctx context.Context,
	tx pgx.Tx,
	postUUID string,
	target model.SocialPostTarget,
	item *model.ContentItem,
	sourceErr error,
	source string,
) (socialPublishWork, model.SocialTargetPublish, *ApiTxError) {
	response := model.SocialTargetPublish{
		PublicationSource: source,
		TargetUuid:        target.Uuid,
		SocialAccountUuid: target.SocialAccountUuid,
		Status:            target.Status,
		RemotePostID:      target.RemotePostID,
	}
	var account model.SocialAccount
	// Hold account metadata stable only until this snapshot/claim commits.
	query := fmt.Sprintf(`
		SELECT uuid, org_uuid, platform, remote_account_id, base_url, enabled
		FROM %s.social_account WHERE uuid = $1 FOR SHARE`,
		h.DbSchema)
	err := tx.QueryRow(ctx, query, target.SocialAccountUuid).Scan(
		&account.Uuid, &account.OrgUuid, &account.Platform, &account.RemoteAccountID, &account.BaseURL, &account.Enabled)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return socialPublishWork{}, response, socialPostDBError(err)
	}
	response.Platform = account.Platform
	task := socialPublishWork{err: sourceErr}
	if task.err == nil {
		task.rendered, task.err = renderSocialTarget(account, *item)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		task.err = fmt.Errorf("social account not found")
	}
	if task.err == nil {
		fingerprint := service.SocialContentFingerprint(task.rendered)
		response.ContentFingerprint = &fingerprint
		// Part 5 successes have no snapshot; do not infer one from today's source.
		var published bool
		err := tx.QueryRow(ctx, fmt.Sprintf(`
			SELECT EXISTS (
			SELECT 1 FROM %[1]s.social_publication
			WHERE social_post_target_uuid = $1 AND status = 'published'
				AND (content_fingerprint = $2 OR content_fingerprint IS NULL))
			OR ($3::boolean AND NOT EXISTS (
				SELECT 1 FROM %[1]s.social_publication
				WHERE social_post_target_uuid = $1 AND status = 'published'))`,
			h.DbSchema), target.Uuid, fingerprint, target.Status == "published" || target.RemotePostID != nil || target.PublishedAt != nil).Scan(&published)
		if err != nil {
			return task, response, socialPostDBError(err)
		}
		if published {
			if txErr := h.completeSocialPublishDuplicateTx(ctx, tx, target.Uuid, fingerprint); txErr != nil {
				return task, response, txErr
			}
			response.Status = "published"
			return task, response, nil
		}
		// Fetch only the access token, into a redacting internal type.
		var secret socialAccountSecret
		if err := tx.QueryRow(ctx, fmt.Sprintf("SELECT access_token FROM %s.social_account WHERE uuid = $1", h.DbSchema), account.Uuid).Scan(&secret.Value); err != nil {
			return task, response, socialPostDBError(err)
		}
		accessToken := ""
		if secret.Value != nil {
			accessToken = *secret.Value
		}
		task.account = service.NewSocialPublishingAccount(account, accessToken)
	}
	query = fmt.Sprintf(`
		UPDATE %s.social_post_target
		SET status = 'publishing', error = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE uuid = $1 AND status = $2`,
		h.DbSchema)
	command, err := tx.Exec(ctx, query, target.Uuid, target.Status)
	if err != nil {
		return task, response, socialPostDBError(err)
	}
	if command.RowsAffected() != 1 {
		return task, response, &ApiTxError{Code: http.StatusConflict, Message: "target is already being published"}
	}
	// A missing account is a preflight failure, never a remote attempt.
	if account.Uuid != "" {
		publicationUuid, err := grains_uuid.Uuidv7String()
		if err != nil {
			return task, response, socialPostDBError(err)
		}
		var text, imageURL, imageAlt *string
		if response.ContentFingerprint != nil {
			text, imageURL, imageAlt = &task.rendered.Text, &task.rendered.ImageURL, task.rendered.ImageAlt
		}
		_, err = tx.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.social_publication
			(uuid, social_post_uuid, social_post_target_uuid, social_account_uuid, platform,
			 content_fingerprint, rendered_text, rendered_image_url, rendered_image_alt, status, publication_source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'publishing', $10)`, h.DbSchema),
			publicationUuid, postUUID, target.Uuid, account.Uuid, account.Platform,
			response.ContentFingerprint, text, imageURL, imageAlt, source)
		if err != nil {
			return task, response, socialPostDBError(err)
		}
		response.PublicationUuid = publicationUuid
	}
	task.claimed = true
	response.Status, response.RemotePostID = "publishing", nil
	return task, response, nil
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
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if result.PublicationUuid != "" {
			status := result.Status
			if status == "publishing" {
				status = "uncertain"
			}
			command, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s.social_publication
				SET status = $2, remote_post_id = $3, error = $4,
					finished_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
				WHERE uuid = $1 AND status = 'publishing'`, h.DbSchema),
				result.PublicationUuid, status, result.RemotePostID, result.Error)
			if err != nil {
				return socialPostDBError(err)
			}
			if command.RowsAffected() != 1 {
				return &ApiTxError{Code: http.StatusConflict, Message: "publication is already resolved"}
			}
		}
		command, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s.social_post_target
			SET status = $2,
				published_at = CASE WHEN $2 = 'published' THEN CURRENT_TIMESTAMP ELSE published_at END,
				remote_post_id = CASE WHEN $2 = 'published' THEN $3 ELSE remote_post_id END,
				error = $4, updated_at = CURRENT_TIMESTAMP
			WHERE uuid = $1 AND status = 'publishing'`, h.DbSchema),
			result.TargetUuid, result.Status, result.RemotePostID, result.Error)
		if err != nil {
			return socialPostDBError(err)
		}
		if command.RowsAffected() != 1 {
			return &ApiTxError{Code: http.StatusConflict, Message: "target is no longer publishing"}
		}
		return nil
	})
	return txErr == nil
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

func (h *ApiHandler) executeSocialPublish(ctx context.Context, task socialPublishWork, target *model.SocialTargetPublish) bool {
	published, err := h.publishSocialTarget(ctx, task)
	target.Status = "published"
	if err == nil {
		target.RemotePostID = &published.RemotePostID
	} else {
		target.Status = "failed"
		if service.SocialPublishUncertain(err) {
			target.Status = "publishing"
		}
		message := err.Error()
		target.Error = &message
	}
	if h.finishSocialPublish(ctx, *target) {
		return true
	}
	target.Status, target.RemotePostID = "publishing", nil
	message := "publication outcome could not be stored; manual reconciliation required"
	target.Error = &message
	return false
}
