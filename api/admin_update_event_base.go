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
	responseType := "admin-update-event-base"

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		JSONError(gc, responseType, http.StatusBadRequest, "eventId is required")
		return
	}

	var payload struct {
		ContentLanguage *string `json:"content_language"`
		Title           string  `json:"title" binding:"required"`
		Subtitle        *string `json:"subtitle,omitempty"`
		Description     *string `json:"description,omitempty"`
		Summary         *string `json:"summary,omitempty"`
	}

	if err := gc.ShouldBindJSON(&payload); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Collect fields to update
	setClauses := []string{"title = $1"} // title is required, always update
	args := []interface{}{payload.Title}
	argPos := 2 // SQL parameter position

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
		JSONError(gc, responseType, txErr.Code, txErr.Error())
		return
	}

	JSONSuccessInfo(gc, responseType)
}
