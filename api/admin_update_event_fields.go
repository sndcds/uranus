package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminUpdateEventFields(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-event-fields")
	ctx := gc.Request.Context()

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Required("eventUuid is required")
		return
	}
	apiRequest.SetMeta("event_uuid", eventUuid)

	var payload struct {
		ReleaseStatus        NullableField[string]   `json:"release_status"`
		ReleaseDate          NullableField[string]   `json:"release_date"`
		ContentLanguage      NullableField[string]   `json:"content_language"`
		Categories           NullableField[[]int]    `json:"categories"`
		Title                NullableField[string]   `json:"title"`
		Subtitle             NullableField[string]   `json:"subtitle"`
		Description          NullableField[string]   `json:"description"`
		Summary              NullableField[string]   `json:"summary"`
		Tags                 NullableField[[]string] `json:"tags"`
		MaxAttendees         NullableField[int]      `json:"max_attendees"`
		MinAge               NullableField[int]      `json:"min_age"`
		MaxAge               NullableField[int]      `json:"max_age"`
		ParticipationInfo    NullableField[string]   `json:"participation_info"`
		PriceType            *string                 `json:"price_type"`
		MinPrice             NullableField[float64]  `json:"min_price"`
		MaxPrice             NullableField[float64]  `json:"max_price"`
		Currency             NullableField[string]   `json:"currency"`
		TicketFlags          *[]string               `json:"ticket_flags"`
		TicketLink           NullableField[string]   `json:"ticket_link"`
		VisitorInfoFlags     NullableField[string]   `json:"visitor_info_flags"`
		RegistrationLink     *string                 `json:"registration_link"`
		RegistrationEmail    *string                 `json:"registration_email"`
		RegistrationPhone    *string                 `json:"registration_phone"`
		RegistrationDeadline *string                 `json:"registration_deadline"`
	}

	decoder := json.NewDecoder(gc.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		debugf(err.Error())
		apiRequest.PayloadError()
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	argPos = addUpdateClauseNullable("release_status", payload.ReleaseStatus, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("release_date", payload.ReleaseDate, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("content_iso_639_1", payload.ContentLanguage, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("categories", payload.Categories, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("title", payload.Title, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("subtitle", payload.Subtitle, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("description", payload.Description, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("summary", payload.Summary, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("tags", payload.Tags, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("max_attendees", payload.MaxAttendees, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("min_age", payload.MinAge, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("max_age", payload.MaxAge, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("participation_info", payload.ParticipationInfo, &setClauses, &args, argPos)
	argPos = addUpdateClauseString("price_type", payload.PriceType, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("min_price", payload.MinPrice, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("max_price", payload.MaxPrice, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("currency", payload.Currency, &setClauses, &args, argPos)
	argPos = addUpdateClauseStringSliceField("ticket_flags", payload.TicketFlags, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("ticket_link", payload.TicketLink, &setClauses, &args, argPos)
	argPos = addUpdateClauseNullable("visitor_info_flags", payload.VisitorInfoFlags, &setClauses, &args, argPos)
	argPos = addUpdateClauseString("registration_link", payload.RegistrationLink, &setClauses, &args, argPos)
	argPos = addUpdateClauseString("registration_email", payload.RegistrationEmail, &setClauses, &args, argPos)
	argPos = addUpdateClauseString("registration_phone", payload.RegistrationPhone, &setClauses, &args, argPos)
	argPos = addUpdateClauseString("registration_deadline", payload.RegistrationDeadline, &setClauses, &args, argPos)

	if len(setClauses) == 0 {
		apiRequest.SuccessNoData(http.StatusOK, "no fields updated")
		return
	}
	apiRequest.SetMeta("field_count", len(setClauses))

	query := fmt.Sprintf(`UPDATE %s.event SET %s WHERE uuid = $%d::uuid`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argPos, // Last placeholder is for WHERE id
	)

	args = append(args, eventUuid) // eventId is the last parameter

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
				Err:  errors.New("event not found"),
			}
		}

		err = RefreshEventProjections(ctx, tx, "event", []string{eventUuid})
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("refresh projection tables failed: %v", err),
			}
		}

		return nil
	})
	if txErr != nil {
		debugf(txErr.Error())
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}
