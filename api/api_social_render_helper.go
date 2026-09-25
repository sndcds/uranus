package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
	"github.com/sndcds/uranus/service"
)

func (h *ApiHandler) socialRenderContent(gc *gin.Context, tx pgx.Tx, post model.SocialPost) (*model.ContentItem, *ApiTxError) {
	item, err := h.loadContentItemTx(gc, tx, post.OrgUuid, post.SourceType, post.SourceUuid, gc.Query("lang"))
	if err != nil {
		return nil, err
	}
	// Without an explicit label language, follow the source language. Both
	// preview and publish use exactly the same normalization and rendering path.
	if _, explicit := gc.Request.URL.Query()["lang"]; !explicit && item.ContentLanguage != nil {
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
