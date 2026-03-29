package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminUpdateEventDates(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grains_api.NewRequest(gc, "admin-update-event-dates")
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
		idsInPayload := []string{}
		for _, d := range payload {
			if d.DateUuid != nil {
				idsInPayload = append(idsInPayload, *d.DateUuid)
			}
		}

		/*
			    TODO: Check!
				if len(idsInPayload) == 0 {
					// No Id´s, delete all dates for this event
					query := fmt.Sprintf(`DELETE FROM %s.event_date WHERE event_id = $1`, h.DbSchema)
					debugf("query: %s", query)
					_, err := tx.Exec(ctx, query, eventId)
					if err != nil {
						debugf(err.Error())
						return &ApiTxError{
							Code: http.StatusInternalServerError,
							Err:  errors.New("delete all event dates failed"),
						}
					}
					return nil
				}
		*/

		query := fmt.Sprintf(
			`DELETE FROM %s.event_date WHERE event_id = $1 AND NOT (id = ANY($2::int[]))`,
			h.DbSchema)
		debugf("query: %s", query)
		debugf("idsInPayload: %s", idsInPayload)
		_, err := tx.Exec(ctx, query, eventUuid, idsInPayload)
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
					debugf("err: %v", err)
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("update date failed: %w", err),
					}
				}
			} else {
				// INSERT
				_, err := tx.Exec(ctx, app.UranusInstance.SqlAdminInsertEventDate,
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
					debugf("err: %v", err)
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("insert date failed: %w", err),
					}
				}
			}
		}

		// Refresh projections
		if err := RefreshEventProjections(ctx, tx, "event", []string{eventUuid}); err != nil {
			debugf("err: %v", err)
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projections failed: %w", err),
			}
		}

		return nil
	})
	if txErr != nil {
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
