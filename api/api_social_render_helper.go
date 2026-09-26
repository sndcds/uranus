package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
	"github.com/sndcds/uranus/service"
)

func (h *ApiHandler) socialRenderContent(gc *gin.Context, tx pgx.Tx, post model.SocialPost) (*model.ContentItem, *ApiTxError) {
	_, explicit := gc.Request.URL.Query()["lang"]
	return h.socialRenderContentTx(gc.Request.Context(), tx, post, gc.Query("lang"), explicit)
}

func (h *ApiHandler) socialRenderContentTx(
	ctx context.Context,
	tx pgx.Tx,
	post model.SocialPost,
	lang string,
	explicit bool,
) (*model.ContentItem, *ApiTxError) {
	item, err := h.loadContentItemTx(ctx, tx, post.OrgUuid, post.SourceType, post.SourceUuid, lang)
	if err != nil {
		return nil, err
	}
	// Without an explicit label language, follow the source language. Both
	// preview and publish use exactly the same normalization and rendering path.
	if !explicit && item.ContentLanguage != nil {
		item.Language = app.NormalizeLocale(*item.ContentLanguage)
	}
	return item, nil
}

func renderSocialTarget(account model.SocialAccount, item model.ContentItem) (model.RenderedPost, error) {
	renderer, err := service.NewSocialRenderer(account.Platform, app.UranusInstance.Config.FrontendClient)
	if err != nil {
		return model.RenderedPost{}, err
	}
	if err = service.ValidateSocialRenderAccount(account, item.OrgUuid); err != nil {
		return model.RenderedPost{}, err
	}
	return renderer.Render(item)
}
