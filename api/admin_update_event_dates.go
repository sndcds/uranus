package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminUpdateEventDates(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-event-dates")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "eventUuid is required")
		return
	}

	type datePayload struct {
		DateUuid  *string `json:"uuid"`
		VenueUuid *string `json:"venue_uuid"`
		SpaceUuid *string `json:"space_uuid"`
		StartDate string  `json:"start_date" binding:"required"`
		StartTime string  `json:"start_time" binding:"required"`
		EndDate   *string `json:"end_date"`
		EndTime   *string `json:"end_time"`
		EntryTime *string `json:"entry_time"`
		Duration  *int    `json:"duration"`
		AllDay    *bool   `json:"all_day"`
	}

	// Wrapper struct to match JSON shape
	var wrapper struct {
		EventDates []datePayload `json:"event_dates"`
	}

	if err := gc.ShouldBindJSON(&wrapper); err != nil {
		apiRequest.PayloadError()
		return
	}

	payload := wrapper.EventDates

	// Validate required fields
	for i, d := range payload {
		if strings.TrimSpace(d.StartDate) == "" {
			apiRequest.Error(http.StatusBadRequest, fmt.Sprintf("start_date is required (index %d)", i))
			return
		}
		if strings.TrimSpace(d.StartTime) == "" {
			apiRequest.Error(http.StatusBadRequest, fmt.Sprintf("start_time is required (index %d)", i))
			return
		}
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		uuidsInPayload := []string{}
		for _, d := range payload {
			if d.DateUuid != nil {
				uuidsInPayload = append(uuidsInPayload, *d.DateUuid)
			}
		}

		query := fmt.Sprintf(
			`DELETE FROM %s.event_date WHERE event_uuid = $1::uuid AND NOT (uuid = ANY($2::uuid[]))`,
			h.DbSchema)
		_, err := tx.Exec(ctx, query, eventUuid, uuidsInPayload)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("delete missing event dates failed: %w", err),
			}
		}

		for _, d := range payload {
			if d.DateUuid != nil {
				// UPDATE
				_, err := tx.Exec(ctx, app.UranusInstance.SqlAdminUpdateEventDate,
					*d.DateUuid,
					eventUuid,
					d.VenueUuid,
					d.SpaceUuid,
					d.StartDate,
					d.StartTime,
					d.EndDate,
					d.EndTime,
					d.EntryTime,
					d.Duration,
					d.AllDay,
					userUuid,
				)
				if err != nil {
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("update date failed: %w", err),
					}
				}
			} else {
				// INSERT
				eventDateUuid, err := grains_uuid.Uuidv7String()
				if err != nil {
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("failed to generate uuid: %v", err),
					}
				}
				_, err = tx.Exec(ctx, app.UranusInstance.SqlAdminInsertEventDate,
					eventDateUuid,
					eventUuid,
					d.VenueUuid,
					d.SpaceUuid,
					d.StartDate,
					d.StartTime,
					d.EndDate,
					d.EndTime,
					d.EntryTime,
					d.Duration,
					d.AllDay,
					userUuid,
				)
				if err != nil {
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("insert date failed: %w", err),
					}
				}
			}
		}

		// Refresh projections
		err = RefreshEventProjections(ctx, tx, "event", []string{eventUuid})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projections failed: %w", err),
			}
		}

		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}

// Convert []int to []interface{} for pgx Exec
func intSliceToInterface(s []int) []interface{} {
	res := make([]interface{}, len(s))
	for i, v := range s {
		res[i] = v
	}
	return res
}

// Generate placeholders for $2, $3, $4… (used in NOT IN)
func pgxPlaceholders(ids []int) string {
	parts := make([]string, len(ids))
	for i := range ids {
		parts[i] = fmt.Sprintf("$%d", i+2) // $1 is eventId
	}
	return strings.Join(parts, ",")
}
