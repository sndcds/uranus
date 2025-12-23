package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Payload for updating title, subtitle, and description
type eventReq struct {
	Title    string  `json:"title" binding:"required"`
	Subtitle *string `json:"subtitle,omitempty"`
}

func (h *ApiHandler) AdminUpdateEventHeader(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	var req eventReq
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build query dynamically depending on whether subtitle is provided
	var query string
	var args []interface{}

	if req.Subtitle != nil {
		query = fmt.Sprintf(`UPDATE %s.event SET title = $2, subtitle = $3 WHERE id = $1`, h.DbSchema)
		args = []interface{}{eventId, req.Title, req.Subtitle}
	} else {
		query = fmt.Sprintf(`UPDATE %s.event SET title = $2 WHERE id = $1`, h.DbSchema)
		args = []interface{}{eventId, req.Title}
	}

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
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":  "event updated successfully",
		"event_id": eventId,
	})
}
