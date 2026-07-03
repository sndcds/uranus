package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) InternalTest(gc *gin.Context) {
	userAgent := strings.ToLower(gc.GetHeader("User-Agent"))
	crawlerFlag := IsCrawler(userAgent)

	eventUuid := gc.Param("eventUuid")
	dateUuid := gc.Param("dateUuid")

	// Load everything via shared function
	event, selectedDate, furtherDates, _ := h.LoadEventByDateUuid(
		gc.Request.Context(),
		eventUuid,
		dateUuid,
		"",
		"de", // TODO: locale via URL
	)

	if crawlerFlag {
		gc.Header("Content-Type", "text/html; charset=utf-8")

		title := event.Title
		description := event.Description

		// Build further dates HTML
		furtherDatesHTML := ""

		for _, d := range furtherDates {
			furtherDatesHTML += fmt.Sprintf(`
			<li>
				<strong>%s</strong>
				%s %s
				– %s
			</li>`,
				d.StartDate,
				d.StartDate,
				d.StartTime,
				d.VenueName,
			)
		}

		selectedDateHTML := ""
		if selectedDate != nil {
			selectedDateHTML = fmt.Sprintf(`
			<div>
				<h3>Selected Date</h3>
				<p>
					<strong>Date:</strong> %s<br>
					<strong>Time:</strong> %s - %s<br>
					<strong>Venue:</strong> %s
				</p>
			</div>`,
				selectedDate.StartDate,
				selectedDate.StartTime,
				selectedDate.EndTime,
				selectedDate.VenueName,
			)
		}

		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
</head>
<body>

    <h1>%s</h1>
    <p>%s</p>

    <hr>

    %s

    <div>
        <h3>Event IDs</h3>
        <strong>eventUuid:</strong> %s<br>
        <strong>dateUuid:</strong> %s<br>
    </div>

    <hr>

    <div>
        <h3>All Dates</h3>
        <ul>
            %s
        </ul>
    </div>

</body>
</html>`,
			title,
			title,
			description,
			selectedDateHTML,
			eventUuid,
			dateUuid,
			furtherDatesHTML,
		)

		gc.String(http.StatusOK, html)
		return
	}

	apiRequest := grains_api.NewRequest(gc, "internal-test")
	apiRequest.Success(http.StatusOK, gin.H{
		"status":     "ok",
		"message":    "internal route works",
		"user_agent": userAgent,
		"is_crawler": crawlerFlag,
	}, "Internal test successful")
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
