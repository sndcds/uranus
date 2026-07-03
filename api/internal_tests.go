package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) InternalTest(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "internal-test")

	apiRequest.Success(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "internal route works",
	}, "Internal test successful")
}

func (h *ApiHandler) InternatGetVenueRelatedEntities(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-venue-related-entities")
	ctx := gc.Request.Context()
	// userUuid := h.userUuid(gc)

	oldVenueUuid := gc.Query("oldVenueUuid")
	if oldVenueUuid == "" {
		apiRequest.Required("oldVenueUuid is required")
		return
	}

	apiRequest.SetMeta("old_venue_uuid", oldVenueUuid)

	type VenueRelated struct {
		EventUuidList     []string
		EventDateUuidList []string
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
		`, oldVenueUuid)

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
