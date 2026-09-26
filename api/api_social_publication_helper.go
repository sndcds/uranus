package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) RegisterSocialPublicationRoutes(admin *gin.RouterGroup) {
	admin.GET("/social/publications", h.AdminGetSocialPublications)
	admin.GET("/social/publications/:uuid", h.AdminGetSocialPublication)
	admin.POST("/social/publications/:uuid/reconcile", h.AdminReconcileSocialPublication)
}

func (h *ApiHandler) socialPublicationQuery() string {
	return fmt.Sprintf(`SELECT p.uuid, p.social_post_uuid, p.social_post_target_uuid, p.social_account_uuid,
		p.platform, p.content_fingerprint, p.rendered_text, p.rendered_image_url, p.rendered_image_alt,
		p.status, p.remote_post_id, p.error, p.started_at, p.finished_at, p.reconciled_at, p.created_at, p.updated_at
		FROM %s.social_publication p`, h.DbSchema)
}

func scanSocialPublication(row pgx.Row) (model.SocialPublication, error) {
	var publication model.SocialPublication
	err := row.Scan(&publication.Uuid, &publication.SocialPostUuid, &publication.SocialPostTargetUuid,
		&publication.SocialAccountUuid, &publication.Platform, &publication.ContentFingerprint,
		&publication.RenderedText, &publication.RenderedImageURL, &publication.RenderedImageAlt,
		&publication.Status, &publication.RemotePostID, &publication.Error, &publication.StartedAt,
		&publication.FinishedAt, &publication.ReconciledAt, &publication.CreatedAt, &publication.UpdatedAt)
	return publication, err
}

func (h *ApiHandler) socialPublicationPermission(gc *gin.Context, tx pgx.Tx, publicationUuid string) *ApiTxError {
	var postUuid string
	err := tx.QueryRow(gc.Request.Context(), fmt.Sprintf(
		"SELECT social_post_uuid FROM %s.social_publication WHERE uuid = $1", h.DbSchema), publicationUuid).Scan(&postUuid)
	if errors.Is(err, pgx.ErrNoRows) {
		return &ApiTxError{Code: http.StatusNotFound, Message: "social publication not found"}
	}
	if err != nil {
		return socialPostDBError(err)
	}
	_, txErr := h.socialPostPermission(gc, tx, postUuid)
	return txErr
}
