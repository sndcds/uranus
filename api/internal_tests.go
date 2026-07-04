package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/model"
	"golang.org/x/net/html"
)

type ShareMeta struct {
	Title       string
	Description string
	ImageUrl    string
	Url         string
	StartTime   *time.Time
	EndTime     *time.Time
	VenueName   string
	VenueCity   string
}

func (h *ApiHandler) InternalTest(gc *gin.Context) {
	eventUuid := gc.Param("eventUuid")
	dateUuid := gc.Param("dateUuid")

	// Load everything via shared function
	event, selectedDate, _, err := h.LoadEventByDateUuid(
		gc.Request.Context(),
		eventUuid,
		dateUuid,
		"",
		"de", // TODO: locale via URL
	)
	if err != nil {
		gc.String(http.StatusNotFound, err.Error())
		return
	}

	if selectedDate == nil {
		log.Println("selectedDate is nil")
	} else {
		log.Printf("selectedDate: %+v", *selectedDate)
	}

	imageURL := ""
	if event.Images != nil {
		if main, ok := event.Images["main"]; ok && main.Uuid != "" {
			imageURL = h.BuildOGImageURL(main.Uuid)
		}
	}

	gc.Header("Content-Type", "text/html; charset=utf-8")

	eventUrl := fmt.Sprintf(
		"https://%s/event/%s/date/%s",
		gc.Request.Host,
		eventUuid,
		dateUuid,
	)

	sm := BuildShareMeta(event, selectedDate, imageURL, eventUrl)
	shareData := struct {
		Share ShareMeta
	}{
		Share: sm,
	}

	if err := h.EventTemplate.Execute(gc.Writer, shareData); err != nil {
		gc.String(http.StatusInternalServerError, err.Error())
	}
	return
}

func (h *ApiHandler) InternalMigrateVenues(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "internal-migrate-venues")
	ctx := gc.Request.Context()
	// userUuid := h.userUuid(gc)

	sourceUuid := gc.Query("source-uuid")
	if sourceUuid == "" {
		apiRequest.Required("source-uuid is required")
		return
	}

	apiRequest.SetMeta("source_uuid", sourceUuid)

	type VenueRelated struct {
		EventUuidList     []string `json:"event_uuid_list"`
		EventDateUuidList []string `json:"event_date_uuid_list"`
	}

	result := VenueRelated{
		EventUuidList:     make([]string, 0),
		EventDateUuidList: make([]string, 0),
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {

		rows, err := tx.Query(ctx, `
			SELECT 'event' AS source, uuid
			FROM uranus.event
			WHERE venue_uuid = $1

			UNION ALL

			SELECT 'event_date' AS source, uuid
			FROM uranus.event_date
			WHERE venue_uuid = $1
		`, sourceUuid)

		if err != nil {
			return TxInternalError(err)
		}
		defer rows.Close()

		for rows.Next() {
			var source string
			var id string

			if err := rows.Scan(&source, &id); err != nil {
				return TxInternalError(err)
			}

			switch source {
			case "event":
				result.EventUuidList = append(result.EventUuidList, id)
			case "event_date":
				result.EventDateUuidList = append(result.EventDateUuidList, id)
			}
		}

		if rows.Err() != nil {
			return TxInternalError(rows.Err())
		}

		return nil
	})

	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Message)
		return
	}

	apiRequest.Success(http.StatusOK, result, "Related venue entities loaded successfully")
}

func IsCrawler(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	bots := []string{
		"googlebot",
		"googleother",
		"adsbot-google",
		"bingbot",
		"duckduckbot",
		"slurp", // Yahoo
		"baiduspider",
		"yandexbot",
		"applebot",
		"facebookexternalhit",
		"twitterbot",
		"linkedinbot",
		"curl",
	}
	for _, bot := range bots {
		if strings.Contains(ua, bot) {
			return true
		}
	}
	return false
}

func BuildShareMeta(event model.EventDetails, date *model.EventDate, imageURL string, url string) ShareMeta {
	sm := ShareMeta{
		Title: event.Title,
		Description: firstNonEmpty(
			deref(event.Summary),
			deref(event.Subtitle)),
		Url: url,
	}

	if date != nil {
		sm.VenueName = deref(date.VenueName)
		sm.VenueCity = deref(date.VenueCity)

		sm.StartTime = combineDateTime(date.StartDate, date.StartTime)
		sm.EndTime = combineDateTime(*date.EndDate, *date.EndTime)
	}

	if imageURL != "" {
		sm.ImageUrl = imageURL
	}

	return sm
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func combineDateTime(dateStr, timeStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	// If no time provided, default to midnight
	if timeStr == "" {
		timeStr = "00:00"
	}

	layout := "2006-01-02 15:04"

	t, err := time.Parse(layout, dateStr+" "+timeStr)
	if err != nil {
		return nil
	}

	return &t
}

func escape(s string) string {
	return html.EscapeString(s)
}

func RenderOG(sm ShareMeta) template.HTML {
	desc := sm.Description

	if sm.StartTime != nil {
		desc = fmt.Sprintf("%s · %s", desc, sm.StartTime.Format("02.01.2006 15:04"))
	}

	if sm.VenueName != "" {
		desc = fmt.Sprintf("%s · %s", desc, sm.VenueName)
	}

	return template.HTML(fmt.Sprintf(`
<meta property="og:type" content="website">
<meta property="og:title" content="%s">
<meta property="og:description" content="%s">
<meta property="og:url" content="%s">
<meta property="og:image" content="%s">
`, escape(sm.Title), escape(desc), sm.Url, sm.ImageUrl))
}

func RenderTwitter(sm ShareMeta) template.HTML {
	return template.HTML(fmt.Sprintf(`
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="%s">
<meta name="twitter:description" content="%s">
<meta name="twitter:image" content="%s">
`, escape(sm.Title), escape(sm.Description), sm.ImageUrl))
}

func RenderJSONLD(sm ShareMeta) template.HTML {
	event := map[string]any{
		"@context":            "https://schema.org",
		"@type":               "Event",
		"name":                sm.Title,
		"description":         sm.Description,
		"image":               sm.ImageUrl,
		"eventAttendanceMode": "https://schema.org/OfflineEventAttendanceMode",
		"eventStatus":         "https://schema.org/EventScheduled",
		"url":                 sm.Url,
	}

	if sm.StartTime != nil {
		event["startDate"] = sm.StartTime.Format(time.RFC3339)
	}

	if sm.EndTime != nil {
		event["endDate"] = sm.EndTime.Format(time.RFC3339)
	}

	if sm.VenueName != "" {
		event["location"] = map[string]any{
			"@type": "Place",
			"name":  sm.VenueName,
			"address": map[string]any{
				"@type":           "PostalAddress",
				"addressLocality": sm.VenueCity,
			},
		}
	}

	b, _ := json.MarshalIndent(event, "", "  ")
	return template.HTML(fmt.Sprintf(`<script type="application/ld+json">%s</script>`, b))
}
