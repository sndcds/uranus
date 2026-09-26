package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminScheduleSocialPost(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-schedule-social-post")
	if !app.ValidUUID(h.userUuid(gc)) {
		apiRequest.Error(http.StatusUnauthorized, "authentication required")
		return
	}
	postUuid, ok := socialPostUUID(gc, apiRequest)
	if !ok {
		return
	}
	var payload struct {
		ScheduledAt time.Time `json:"scheduled_at"`
	}
	if err := grains_api.BindJSONStrict(gc, &payload); err != nil {
		apiRequest.PayloadError()
		return
	}
	if !payload.ScheduledAt.After(time.Now()) {
		apiRequest.Error(http.StatusBadRequest, "scheduled_at must be in the future")
		return
	}

	result, txErr := h.updateSocialSchedule(gc, postUuid, &payload.ScheduledAt)
	if txErr != nil {
		socialPostRespondError(apiRequest, txErr)
		return
	}
	apiRequest.Success(http.StatusOK, result)
}

func (h *ApiHandler) AdminCancelSocialPost(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-cancel-social-post")
	if !app.ValidUUID(h.userUuid(gc)) {
		apiRequest.Error(http.StatusUnauthorized, "authentication required")
		return
	}
	postUuid, ok := socialPostUUID(gc, apiRequest)
	if !ok {
		return
	}

	result, txErr := h.updateSocialSchedule(gc, postUuid, nil)
	if txErr != nil {
		socialPostRespondError(apiRequest, txErr)
		return
	}
	apiRequest.Success(http.StatusOK, result)
}

// A nil timestamp cancels the post's scheduled targets. Both actions use the
// same post lock as publishing and CRUD; neither can undo a committed claim.
func (h *ApiHandler) updateSocialSchedule(
	gc *gin.Context,
	postUuid string,
	scheduledAt *time.Time,
) (model.SocialPost, *ApiTxError) {
	ctx := gc.Request.Context()
	var result model.SocialPost
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if _, txErr := h.socialPostPermission(gc, tx, postUuid); txErr != nil {
			return txErr
		}
		if txErr := h.socialPostPublishingConflict(gc, tx, postUuid); txErr != nil {
			return txErr
		}
		query := fmt.Sprintf(`
			UPDATE %s.social_post_target
			SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP
			WHERE social_post_uuid = $1 AND status = 'scheduled'`,
			h.DbSchema)
		args := []any{postUuid}
		if scheduledAt != nil {
			// Recheck after acquiring the post lock, which may have waited.
			var future bool
			if err := tx.QueryRow(ctx, "SELECT $1::timestamptz > clock_timestamp()", scheduledAt).Scan(&future); err != nil {
				return socialPostDBError(err)
			}
			if !future {
				return &ApiTxError{Code: http.StatusBadRequest, Message: "scheduled_at must be in the future"}
			}
			query = fmt.Sprintf(`
				UPDATE %[1]s.social_post_target t
				SET status = 'scheduled', scheduled_at = $2, publication_source = 'scheduled',
					publish_language = NULL, error = NULL, updated_at = CURRENT_TIMESTAMP
				WHERE social_post_uuid = $1 AND status IN ('draft', 'failed', 'scheduled', 'published')
					AND (status <> 'published' OR remote_post_id IS NOT NULL OR published_at IS NOT NULL
						OR EXISTS (SELECT 1 FROM %[1]s.social_publication p
							WHERE p.social_post_target_uuid = t.uuid AND p.status = 'published'))`,
				h.DbSchema)
			args = append(args, scheduledAt)
		}
		command, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return socialPostDBError(err)
		}
		if command.RowsAffected() == 0 {
			return &ApiTxError{Code: http.StatusConflict, Message: "social post has no eligible targets"}
		}
		result, err = scanSocialPost(tx.QueryRow(ctx, h.socialPostQuery()+" WHERE p.uuid = $1", postUuid))
		if err != nil {
			return socialPostDBError(err)
		}
		return nil
	})
	return result, txErr
}
