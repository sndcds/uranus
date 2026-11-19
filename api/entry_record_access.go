package api

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type RequestSource string

const (
	RequestSourceFrontend RequestSource = "frontend"
	RequestSourceDirect   RequestSource = "direct"
	RequestSourceCrawler  RequestSource = "crawler"
	RequestSourceExternal RequestSource = "external"
	RequestSourceUnknown  RequestSource = "unknown"
)

func (h *ApiHandler) RecordEntryAccess(
	gc *gin.Context,
	contextType string,
	entryUuid string,
	subUuid string,
	lang string,
) error {

	referrer := GetReferrer(gc)
	source := GetRequestSource(gc)

	const query = `
        INSERT INTO uranus.entry_access (
            context,
            entry_uuid,
            sub_uuid,
            lang,
            referrer,
			source
        )
        VALUES (
            $1,
            $2::uuid,
            NULLIF($3, '')::uuid,
            NULLIF($4, ''),
            NULLIF($5, ''),
            NULLIF($6, '')
        )
    `

	_, err := h.DbPool.Exec(
		gc.Request.Context(),
		query,
		contextType,
		entryUuid,
		subUuid,
		lang,
		referrer,
		source,
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

func GetRequestSource(gc *gin.Context) RequestSource {
	origin := strings.ToLower(gc.GetHeader("Origin"))
	ua := strings.ToLower(gc.GetHeader("User-Agent"))
	fetchSite := gc.GetHeader("Sec-Fetch-Site")
	fetchMode := gc.GetHeader("Sec-Fetch-Mode")

	// Known kulturbytes frontend.
	parsed, err := url.Parse(origin)
	if err == nil && parsed.Scheme == "https" {
		switch parsed.Hostname() {
		case "app.kulturbytes.de", "kulturbytes.de":
			return RequestSourceFrontend
		}
	}

	// Search engine crawlers and other known bots.
	bots := []string{
		"googlebot",
		"bingbot",
		"duckduckbot",
		"yandexbot",
		"baiduspider",
		"facebookexternalhit",
		"twitterbot",
		"linkedinbot",
	}

	for _, bot := range bots {
		if strings.Contains(ua, bot) {
			return RequestSourceCrawler
		}
	}

	// Browser navigation directly to the API.
	if fetchSite == "none" && fetchMode == "navigate" {
		return RequestSourceDirect
	}

	// Requests from other websites.
	if origin != "" {
		return RequestSourceExternal
	}

	return RequestSourceUnknown
}
