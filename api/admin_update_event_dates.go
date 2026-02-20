package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminUpdateEventDates(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiResponseType := "admin-update-event-dates"
	userId := h.userId(gc)

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		JSONError(gc, apiResponseType, http.StatusBadRequest, "eventId is required")
		return
	}

	type datePayload struct {
		DateId    *int    `json:"id"`
		VenueId   *int    `json:"venue_id"`
		SpaceId   *int    `json:"space_id"`
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
		JSONError(gc, apiResponseType, http.StatusBadRequest, err.Error())
		return
	}

	payload := wrapper.EventDates

	// Validate required fields
	for i, d := range payload {
		if strings.TrimSpace(d.StartDate) == "" {
			JSONError(gc, apiResponseType, http.StatusBadRequest,
				fmt.Sprintf("start_date is required (index %d)", i))
			return
		}
		if strings.TrimSpace(d.StartTime) == "" {
			JSONError(gc, apiResponseType, http.StatusBadRequest,
				fmt.Sprintf("start_time is required (index %d)", i))
			return
		}
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		idsInPayload := []int{}
		for _, d := range payload {
			if d.DateId != nil {
				idsInPayload = append(idsInPayload, *d.DateId)
			}
		}

		/*
			if len(idsInPayload) == 0 {
				// No Id´s, delete all dates for this event
				query := fmt.Sprintf(`DELETE FROM %s.event_date WHERE event_id = $1`, h.DbSchema)
				debugf("query: %s", query)
				_, err := tx.Exec(ctx, query, eventId)
				if err != nil {
					debugf(err.Error())
					return &ApiTxError{
						Code: http.StatusInternalServerError,
						Err:  fmt.Errorf("delete all event dates failed"),
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
		_, err := tx.Exec(ctx, query, eventId, idsInPayload)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("delete missing event dates failed: %w", err),
			}
		}

		for _, d := range payload {
			if d.DateId != nil {
				// UPDATE
				_, err := tx.Exec(ctx, app.UranusInstance.SqlAdminUpdateEventDate,
					*d.DateId,
					eventId,
					d.VenueId,
					d.SpaceId,
					d.StartDate,
					d.StartTime,
					d.EndDate,
					d.EndTime,
					d.EntryTime,
					d.Duration,
					d.AllDay,
					userId,
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
					eventId,
					d.VenueId,
					d.SpaceId,
					d.StartDate,
					d.StartTime,
					d.EndDate,
					d.EndTime,
					d.EntryTime,
					d.Duration,
					d.AllDay,
					userId,
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
		if err := RefreshEventProjections(ctx, tx, "event", []int{eventId}); err != nil {
			debugf("err: %v", err)
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projections failed: %w", err),
			}
		}

		return nil
	})
	if txErr != nil {
		JSONError(gc, apiResponseType, txErr.Code, txErr.Error())
		return
	}

	JSONSuccessNoData(gc, apiResponseType)
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
