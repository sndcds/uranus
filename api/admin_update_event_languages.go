package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h *ApiHandler) AdminUpdateEventLanguages(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiResponseType := "admin-update-event-languages"

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		JSONError(gc, apiResponseType, http.StatusBadRequest, "eventId is required")
		return
	}

	type eventLanguagesReq struct {
		Languages []string `json:"languages" binding:"required"`
	}

	var req eventLanguagesReq
	if err := gc.ShouldBindJSON(&req); err != nil {
		JSONPayloadError(gc, apiResponseType)
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		query := fmt.Sprintf(`UPDATE %s.event SET languages = $2 WHERE id = $1`, h.DbSchema)

		res, err := tx.Exec(ctx, query, eventId, req.Languages)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to update event: %v", err),
			}
		}

		rowsAffected := res.RowsAffected()
		if rowsAffected == 0 {
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
		JSONDatabaseError(gc, apiResponseType)
		return
	}

	JSONSuccessNoData(gc, apiResponseType)
}
