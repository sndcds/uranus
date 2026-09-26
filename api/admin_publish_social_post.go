package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminPublishSocialPost(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-publish-social-post")
	ctx := gc.Request.Context()
	if !app.ValidUUID(h.userUuid(gc)) {
		apiRequest.Error(http.StatusUnauthorized, "authentication required")
		return
	}
	postUuid, ok := socialPostUUID(gc, apiRequest)
	if !ok {
		return
	}
	var publishLanguage *string
	if _, explicit := gc.Request.URL.Query()["lang"]; explicit {
		lang := app.NormalizeLocale(gc.Query("lang"))
		publishLanguage = &lang
	}

	result := model.SocialPostPublish{PostUuid: postUuid, Results: []model.SocialTargetPublish{}}
	queued := false
	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if _, txErr := h.socialPostPermission(gc, tx, postUuid); txErr != nil {
			return txErr
		}
		post, err := scanSocialPost(tx.QueryRow(ctx, h.socialPostQuery()+" WHERE p.uuid = $1", postUuid))
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
		for _, target := range post.Targets {
			response := model.SocialTargetPublish{
				TargetUuid: target.Uuid, SocialAccountUuid: target.SocialAccountUuid,
				Status: target.Status, RemotePostID: target.RemotePostID,
				PublicationSource: target.PublicationSource,
			}
			err := tx.QueryRow(ctx, fmt.Sprintf(
				"SELECT platform FROM %s.social_account WHERE uuid = $1", h.DbSchema),
				target.SocialAccountUuid).Scan(&response.Platform)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return socialPostDBError(err)
			}
			if busy || (target.Status != "draft" && target.Status != "failed" && target.Status != "published") {
				message := "target cannot be published from status " + target.Status
				if busy {
					message = "social post has an unresolved publishing target"
				}
				response.Error = &message
				result.Results = append(result.Results, response)
				continue
			}
			if target.Status == "published" && target.RemotePostID == nil && target.PublishedAt == nil {
				var recorded bool
				query := fmt.Sprintf(`
					SELECT EXISTS (SELECT 1 FROM %s.social_publication
						WHERE social_post_target_uuid = $1 AND status = 'published')`,
					h.DbSchema)
				if err := tx.QueryRow(ctx, query, target.Uuid).Scan(&recorded); err != nil {
					return socialPostDBError(err)
				}
				if !recorded {
					message := "legacy published snapshot is unavailable"
					response.Error = &message
					result.Results = append(result.Results, response)
					continue
				}
			}
			// Only enqueue here. The separate worker renders and contacts the platform.
			query := fmt.Sprintf(`
				UPDATE %s.social_post_target
				SET status = 'scheduled', scheduled_at = CURRENT_TIMESTAMP,
					publication_source = 'manual', publish_language = $2,
					error = NULL, updated_at = CURRENT_TIMESTAMP
				WHERE uuid = $1 AND status = $3`,
				h.DbSchema)
			command, err := tx.Exec(ctx, query, target.Uuid, publishLanguage, target.Status)
			if err != nil {
				return socialPostDBError(err)
			}
			if command.RowsAffected() != 1 {
				return &ApiTxError{Code: http.StatusConflict, Message: "target is already queued or publishing"}
			}
			queued = true
			response.Status, response.PublicationSource, response.RemotePostID = "scheduled", "manual", nil
			result.Results = append(result.Results, response)
		}
		return nil
	})
	if txErr != nil {
		socialPostRespondError(apiRequest, txErr)
		return
	}
	status := http.StatusAccepted
	if !queued {
		status = http.StatusConflict
	}
	apiRequest.Success(status, result)
}
