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

	fmt.Println("eventUuid: %s, dateUuid: %s", eventUuid, dateUuid)

	if crawlerFlag {
		gc.Header("Content-Type", "text/html; charset=utf-8")
		gc.String(http.StatusOK, `<!DOCTYPE html>
<html>
<head>
    <title>Internal Test</title>
</head>
<body>
    <h1>80s Party</h1>
    <p>With DJ Milligramm.</p>
</body>
</html>`)
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
