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

	origin := GetRequestHeader(gc, "Origin")
	userAgent := GetRequestHeader(gc, "User-Agent")
	acceptLanguage := GetRequestHeader(gc, "Accept-Language")

	secFetchSite := GetRequestHeader(gc, "Sec-Fetch-Site")
	secFetchMode := GetRequestHeader(gc, "Sec-Fetch-Mode")
	secFetchDest := GetRequestHeader(gc, "Sec-Fetch-Dest")

	const query = `
		INSERT INTO uranus.entry_access (
			context,
			entry_uuid,
			sub_uuid,
			lang,
			referrer,
			source,
			request_mode,
			origin,
			user_agent,
			accept_language,
			sec_fetch_site,
			sec_fetch_mode,
			sec_fetch_dest
		)
		VALUES (
			$1,
			$2::uuid,
			NULLIF($3, '')::uuid,
			NULLIF($4, ''),
			NULLIF($5, ''),
			NULLIF($6, ''),
			NULLIF($7, ''),
			NULLIF($8, ''),
			NULLIF($9, ''),
			NULLIF($10, ''),
			NULLIF($11, ''),
			NULLIF($12, ''),
			NULLIF($13, '')
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
		origin,
		userAgent,
		acceptLanguage,
		secFetchSite,
		secFetchMode,
		secFetchDest,
	)

	return err
}

func GetRequestHeader(gc *gin.Context, name string) string {
	return strings.TrimSpace(gc.GetHeader(name))
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
	origin := strings.ToLower(GetRequestHeader(gc, "Origin"))
	ua := strings.ToLower(GetRequestHeader(gc, "User-Agent"))

	requestSource := GetRequestHeader(gc, "X-Kulturbytes-Request-Source")

	if requestSource == "frontend-ssr" {
		return RequestSourceFrontend
	}

	parsed, err := url.Parse(origin)
	if err == nil && parsed.Scheme == "https" {
		switch parsed.Hostname() {
		case "app.kulturbytes.de", "kulturbytes.de":
			return RequestSourceFrontend
		}
	}

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

	fetchSite := GetRequestHeader(gc, "Sec-Fetch-Site")
	fetchMode := GetRequestHeader(gc, "Sec-Fetch-Mode")

	if fetchSite == "none" && fetchMode == "navigate" {
		return RequestSourceDirect
	}

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
