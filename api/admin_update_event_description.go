package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type eventDescriptionRequest struct {
	Description string `json:"description" binding:"required"`
}

func (h *ApiHandler) AdminUpdateEventDescription(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	var req eventDescriptionRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		query := fmt.Sprintf(`UPDATE %s.event SET description = $2 WHERE id = $1`, h.DbSchema)

		res, err := h.DbPool.Exec(ctx, query, eventId, req.Description)
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
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":  "event description updated successfully",
		"event_id": eventId,
	})
}
