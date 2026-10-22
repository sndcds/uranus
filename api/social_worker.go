package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) RunSocialWorker(ctx context.Context, once bool) error {
	interval := h.Config.SocialWorkerInterval
	batchSize := h.Config.SocialWorkerBatchSize
	if interval < 15 || interval > 3600 || batchSize < 1 || batchSize > 100 {
		return fmt.Errorf("social worker requires interval 15-3600 seconds and batch size 1-100")
	}
	log.Print("Uranus: social worker started")
	defer log.Print("Uranus: social worker stopped")

	for {
		if ctx.Err() != nil {
			return nil
		}
		_, err := h.ProcessDueSocialTargets(ctx, batchSize)
		if ctx.Err() != nil {
			return nil
		}
		if once {
			return err
		}
		if err != nil {
			log.Print("Uranus: social worker database operation failed")
		}
		// Wait after every bounded batch, including an empty queue or DB error.
		timer := time.NewTimer(time.Duration(interval) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (h *ApiHandler) ProcessDueSocialTargets(ctx context.Context, batchSize int) (int, error) {
	if batchSize < 1 || batchSize > 100 {
		return 0, fmt.Errorf("social worker batch size must be 1-100")
	}
	processed := 0
	for processed < batchSize && ctx.Err() == nil {
		task, result, txErr := h.claimScheduledSocialTarget(ctx)
		if txErr != nil {
			return processed, fmt.Errorf("social worker database operation failed")
		}
		if result.TargetUuid == "" {
			break
		}
		processed++
		if !task.claimed {
			log.Printf("Uranus: social target %s already published", result.TargetUuid)
			continue
		}
		log.Printf("Uranus: social target %s claimed, publication %s", result.TargetUuid, result.PublicationUuid)
		if !h.executeSocialPublish(ctx, task, &result) {
			log.Printf("Uranus: social publication %s requires reconciliation", result.PublicationUuid)
			continue
		}
		log.Printf("Uranus: social publication %s target status %s", result.PublicationUuid, result.Status)
	}
	return processed, nil
}

func (h *ApiHandler) claimScheduledSocialTarget(ctx context.Context) (socialPublishWork, model.SocialTargetPublish, *ApiTxError) {
	var task socialPublishWork
	var result model.SocialTargetPublish
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Lock posts first, as manual publishing and CRUD do. SKIP LOCKED lets
		// multiple workers claim other posts without waiting for one another.
		var postUuid string
		query := fmt.Sprintf(`
			SELECT p.uuid
			FROM %[1]s.social_post_target t
			JOIN %[1]s.social_post p ON p.uuid = t.social_post_uuid
			WHERE t.status = 'scheduled' AND t.scheduled_at <= CURRENT_TIMESTAMP
				AND NOT EXISTS (SELECT 1 FROM %[1]s.social_post_target busy
					WHERE busy.social_post_uuid = p.uuid AND busy.status = 'publishing')
			ORDER BY t.scheduled_at, t.created_at, t.uuid
			LIMIT 1 FOR UPDATE OF p SKIP LOCKED`,
			h.DbSchema)
		err := tx.QueryRow(ctx, query).Scan(&postUuid)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return socialPostDBError(err)
		}
		// Recheck current state after the post lock; scheduling may have
		// committed between the candidate snapshot and acquiring that lock.
		var targetUuid string
		query = fmt.Sprintf(`
			SELECT t.uuid FROM %[1]s.social_post_target t
			WHERE t.social_post_uuid = $1 AND t.status = 'scheduled'
				AND t.scheduled_at <= CURRENT_TIMESTAMP
				AND NOT EXISTS (SELECT 1 FROM %[1]s.social_post_target busy
					WHERE busy.social_post_uuid = $1 AND busy.status = 'publishing')
			ORDER BY t.scheduled_at, t.created_at, t.uuid
			LIMIT 1 FOR UPDATE OF t`,
			h.DbSchema)
		err = tx.QueryRow(ctx, query, postUuid).Scan(&targetUuid)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return socialPostDBError(err)
		}
		post, err := scanSocialPost(tx.QueryRow(ctx, h.socialPostQuery()+" WHERE p.uuid = $1", postUuid))
		if err != nil {
			return socialPostDBError(err)
		}
		for _, target := range post.Targets {
			if target.Uuid != targetUuid {
				continue
			}
			lang := ""
			if target.PublishLanguage != nil {
				lang = *target.PublishLanguage
			}
			item, sourceErr := h.socialRenderContentTx(ctx, tx, post, lang, target.PublishLanguage != nil)
			if sourceErr != nil && sourceErr.Code >= http.StatusInternalServerError {
				return sourceErr
			}
			var localErr error
			if sourceErr != nil {
				localErr = errors.New(sourceErr.Error())
			}
			var txErr *ApiTxError
			task, result, txErr = h.prepareSocialPublishTx(ctx, tx, postUuid, target, item, localErr, target.PublicationSource)
			return txErr
		}
		return nil
	})
	return task, result, txErr
}

func (h *ApiHandler) completeSocialPublishDuplicateTx(ctx context.Context, tx pgx.Tx, targetUuid, fingerprint string) *ApiTxError {
	// Restore the matching historical result, including when content reverted
	// to an older fingerprint. Legacy successes retain their current metadata.
	query := fmt.Sprintf(`
		UPDATE %[1]s.social_post_target t
		SET status = 'published', error = NULL, updated_at = CURRENT_TIMESTAMP,
			remote_post_id = COALESCE((SELECT p.remote_post_id FROM %[1]s.social_publication p
				WHERE p.social_post_target_uuid = t.uuid AND p.status = 'published'
					AND (p.content_fingerprint = $2 OR p.content_fingerprint IS NULL)
				ORDER BY p.created_at DESC, p.uuid DESC LIMIT 1), t.remote_post_id),
			published_at = COALESCE((SELECT p.finished_at FROM %[1]s.social_publication p
				WHERE p.social_post_target_uuid = t.uuid AND p.status = 'published'
					AND (p.content_fingerprint = $2 OR p.content_fingerprint IS NULL)
				ORDER BY p.created_at DESC, p.uuid DESC LIMIT 1), t.published_at)
		WHERE t.uuid = $1 AND t.status = 'scheduled'`,
		h.DbSchema)
	if _, err := tx.Exec(ctx, query, targetUuid, fingerprint); err != nil {
		return socialPostDBError(err)
	}
	return nil
}
