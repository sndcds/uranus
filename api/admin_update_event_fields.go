package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grainsapi"
)

func (h *ApiHandler) UpdateEventFields(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiRequest := grainsapi.NewRequest(gc, "admin-update-event-fields")

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "eventId is required")
		return
	}

	var payload struct {
		ReleaseStatus     NullableField[string]   `json:"release_status"`
		ReleaseDate       NullableField[string]   `json:"release_date"`
		ContentLanguage   NullableField[string]   `json:"content_language"`
		Title             NullableField[string]   `json:"title"`
		Subtitle          NullableField[string]   `json:"subtitle"`
		Description       NullableField[string]   `json:"description"`
		Summary           NullableField[string]   `json:"summary"`
		Tags              NullableField[[]string] `json:"tags"`
		MaxAttendees      NullableField[int]      `json:"max_attendees"`
		MinAge            NullableField[int]      `json:"min_age"`
		MaxAge            NullableField[int]      `json:"max_age"`
		ParticipationInfo NullableField[string]   `json:"participation_info"`
		PriceType         *string                 `json:"price_type"`
		MinPrice          NullableField[float64]  `json:"min_price"`
		MaxPrice          NullableField[float64]  `json:"max_price"`
		Currency          NullableField[string]   `json:"currency"`
		TicketFlags       *[]string               `json:"ticket_flags"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	argPos = addUpdateClauseNullable("release_status", payload.ReleaseStatus, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("release_date", payload.ReleaseDate, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("content_iso_639_1", payload.ContentLanguage, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("title", payload.Title, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("subtitle", payload.Subtitle, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("description", payload.Description, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("summary", payload.Summary, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("tags", payload.Tags, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("max_attendees", payload.MaxAttendees, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("min_age", payload.MinAge, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("max_age", payload.MaxAge, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("participation_info", payload.ParticipationInfo, &setClauses, &args, argPos)
	argPos = addStringUpdateClause("price_type", payload.PriceType, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("min_price", payload.MinPrice, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("max_price", payload.MaxPrice, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("currency", payload.Currency, &setClauses, &args, argPos)
	argPos = addUpdateStringSliceField("ticket_flags", payload.TicketFlags, &setClauses, &args, argPos)

	if len(setClauses) == 0 {
		apiRequest.SuccessNoData(http.StatusOK, "no fields updated")
		return
	}
	fmt.Println("len(setClauses)", len(setClauses))

	query := fmt.Sprintf(`UPDATE %s.event SET %s WHERE id = $%d`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argPos, // Last placeholder is for WHERE id
	)
	fmt.Println("query", query)

	args = append(args, eventId) // eventId is the last parameter
	fmt.Println("args", args)

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		res, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to update event: %v", err),
			}
		}

		if res.RowsAffected() == 0 {
			return &ApiTxError{
				Code: http.StatusNotFound,
				Err:  fmt.Errorf("event not found"),
			}
		}

		err = RefreshEventProjections(ctx, tx, "event", []int{eventId})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %v", err),
			}
		}

		return nil
	})
	if txErr != nil {
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}
