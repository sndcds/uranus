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

type RequestMode string

const (
	RequestModeSSR     RequestMode = "ssr"
	RequestModeClient  RequestMode = "client"
	RequestModeUnknown RequestMode = "unknown"
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
	requestMode := GetRequestMode(gc)

	const query = `
        INSERT INTO uranus.entry_access (
            context,
            entry_uuid,
            sub_uuid,
            lang,
            referrer,
            source,
            request_mode
        )
        VALUES (
            $1,
            $2::uuid,
            NULLIF($3, '')::uuid,
            NULLIF($4, ''),
            NULLIF($5, ''),
            NULLIF($6, ''),
            NULLIF($7, '')
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
		requestMode,
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

	// Explicit Kulturbytes frontend identification
	requestSource := gc.GetHeader("X-Kulturbytes-Request-Source")

	if requestSource == "frontend-ssr" {
		return RequestSourceFrontend
	}

	// Known Kulturbytes frontend
	parsed, err := url.Parse(origin)
	if err == nil && parsed.Scheme == "https" {
		switch parsed.Hostname() {
		case "app.kulturbytes.de", "kulturbytes.de":
			return RequestSourceFrontend
		}
	}

	// Search engine crawlers and other known bots
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

	fetchSite := gc.GetHeader("Sec-Fetch-Site")
	fetchMode := gc.GetHeader("Sec-Fetch-Mode")

	// Browser navigation directly to the API
	if fetchSite == "none" && fetchMode == "navigate" {
		return RequestSourceDirect
	}

	// Requests from other websites
	if origin != "" {
		return RequestSourceExternal
	}

	return RequestSourceUnknown
}

func GetRequestMode(gc *gin.Context) RequestMode {
	switch gc.Query("request_mode") {
	case string(RequestModeSSR):
		return RequestModeSSR

	case string(RequestModeClient):
		return RequestModeClient

	default:
		return RequestModeUnknown
	}
}
