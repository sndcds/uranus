package api

import (
	"context"
	"net/url"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) RecordEntryAccess(
	ctx context.Context,
	contextType string,
	entryUuid string,
	subUuid string,
	lang string,
	referrer string,
) error {

	const query = `
        INSERT INTO uranus.entry_access (
            context,
            entry_uuid,
            sub_uuid,
            lang,
            referrer
        )
        VALUES (
            $1,
            $2::uuid,
            NULLIF($3, '')::uuid,
            NULLIF($4, ''),
            NULLIF($5, '')
        )
    `

	_, err := h.DbPool.Exec(
		ctx,
		query,
		contextType,
		entryUuid,
		subUuid,
		lang,
		referrer,
	)

	return err
}

func GetReferrer(gc *gin.Context) string {
	referrer := gc.Request.Referer()
	if referrer == "" {
		return ""
	}

	u, err := url.Parse(referrer)
	if err != nil {
		return ""
	}

	return u.Scheme + "://" + u.Host
}
