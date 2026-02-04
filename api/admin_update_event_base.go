package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h *ApiHandler) AdminUpdateEventBase(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiResponseType := "admin-update-event-base"

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		JSONError(gc, apiResponseType, http.StatusBadRequest, "eventId is required")
		return
	}

	var payload struct {
		ContentLanguage *string `json:"content_language"`
		Title           *string `json:"title,omitempty"`
		Subtitle        *string `json:"subtitle,omitempty"`
		Description     *string `json:"description,omitempty"`
		Summary         *string `json:"summary,omitempty"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		JSONPayloadError(gc, apiResponseType)
		return
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if payload.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argPos))
		args = append(args, *payload.Title)
		argPos++
	}

	if payload.ContentLanguage != nil {
		setClauses = append(setClauses, fmt.Sprintf("content_iso_639_1 = $%d", argPos))
		args = append(args, *payload.ContentLanguage)
		argPos++
	}

	if payload.Subtitle != nil {
		setClauses = append(setClauses, fmt.Sprintf("subtitle = $%d", argPos))
		args = append(args, *payload.Subtitle)
		argPos++
	}

	if payload.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *payload.Description)
		argPos++
	}

	if payload.Summary != nil {
		setClauses = append(setClauses, fmt.Sprintf("summary = $%d", argPos))
		args = append(args, *payload.Summary)
		argPos++
	}

	query := fmt.Sprintf(`UPDATE %s.event SET %s WHERE id = $%d`,
		h.DbSchema,
		strings.Join(setClauses, ", "),
		argPos, // Last placeholder is for WHERE id
	)

	args = append(args, eventId) // eventId is the last parameter
	fmt.Println("query", query)
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
		JSONError(gc, apiResponseType, txErr.Code, txErr.Error())
		return
	}

	JSONSuccessNoData(gc, apiResponseType)
}
