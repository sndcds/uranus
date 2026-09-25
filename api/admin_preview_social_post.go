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
	"github.com/sndcds/uranus/service"
)

// AdminPreviewSocialPost reads current content and account metadata only. All
// results are buffered until the transaction succeeds; no partial preview is sent.
func (h *ApiHandler) AdminPreviewSocialPost(gc *gin.Context) {
	request := grains_api.NewRequest(gc, "admin-preview-social-post")
	if !app.ValidUUID(h.userUuid(gc)) {
		request.Error(http.StatusUnauthorized, "authentication required")
		return
	}
	postUUID, ok := socialPostUUID(gc, request)
	if !ok {
		return
	}
	result := model.SocialPostPreview{PostUuid: postUUID, Previews: []model.SocialTargetPreview{}}
	txErr := WithTransaction(gc.Request.Context(), h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Include the source relations and account metadata in one consistent snapshot.
		if _, err := tx.Exec(gc.Request.Context(), "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ"); err != nil {
			return socialPostDBError(err)
		}
		orgUUID, err := h.socialPostPermission(gc, tx, postUUID)
		if err != nil {
			return err
		}
		post, dbErr := scanSocialPost(tx.QueryRow(gc.Request.Context(), h.socialPostQuery()+" WHERE p.uuid = $1", postUUID))
		if dbErr != nil {
			return socialPostDBError(dbErr)
		}
		if len(post.Targets) == 0 {
			return &ApiTxError{Code: http.StatusBadRequest, Message: "social post has no targets"}
		}
		item, err := h.loadContentItemTx(gc, tx, orgUUID, post.SourceType, post.SourceUuid, gc.Query("lang"))
		if err != nil {
			return err
		}
		// Source text is not translated in Uranus. With no explicit label language,
		// follow the content language and the existing supported-language fallback.
		if _, explicit := gc.Request.URL.Query()["lang"]; !explicit && item.ContentLanguage != nil {
			item.Language = app.NormalizeLocale(*item.ContentLanguage)
		}
		for _, target := range post.Targets {
			// Deliberately omit credentials, token expiry and publication metadata.
			var account model.SocialAccount
			dbErr := tx.QueryRow(gc.Request.Context(), fmt.Sprintf(`SELECT uuid, org_uuid, platform, remote_account_id, base_url, enabled
    FROM %s.social_account WHERE uuid = $1`, h.DbSchema), target.SocialAccountUuid).Scan(
				&account.Uuid, &account.OrgUuid, &account.Platform, &account.RemoteAccountID, &account.BaseURL, &account.Enabled)
			fail := func(reason string) *ApiTxError {
				return &ApiTxError{Code: http.StatusBadRequest, Message: fmt.Sprintf("target %s (platform %s): %s", target.Uuid, account.Platform, reason)}
			}
			if errors.Is(dbErr, pgx.ErrNoRows) {
				return fail("social account not found")
			}
			if dbErr != nil {
				return socialPostDBError(dbErr)
			}
			renderer, renderErr := service.NewSocialRenderer(account.Platform, app.UranusInstance.Config.FrontendClient)
			if renderErr != nil {
				return fail(renderErr.Error())
			}
			if renderErr = service.ValidateSocialRenderAccount(account, orgUUID); renderErr != nil {
				return fail(renderErr.Error())
			}
			rendered, renderErr := renderer.Render(*item)
			if renderErr != nil {
				return fail(renderErr.Error())
			}
			result.Previews = append(result.Previews, model.SocialTargetPreview{TargetUuid: target.Uuid, RenderedPost: rendered})
		}
		return nil
	})
	if txErr != nil {
		socialPostRespondError(request, txErr)
		return
	}
	request.Success(http.StatusOK, result)
}
