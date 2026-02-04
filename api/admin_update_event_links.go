package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (h *ApiHandler) AdminUpdateEventLinks(gc *gin.Context) {
	ctx := gc.Request.Context()
	apiResponseType := "admin-update-event-urls"

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		JSONError(gc, apiResponseType, http.StatusBadRequest, "eventId is required")
		return
	}

	type eventLinksRequest struct {
		Types []struct {
			Label *string `json:"label"`
			Type  int     `json:"type" binding:"required"`
			Url   string  `json:"url" binding:"required"`
		} `json:"event_links" binding:"required"`
	}

	var req eventLinksRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Delete existing type-genre links
		deleteQuery := fmt.Sprintf(`DELETE FROM %s.event_link WHERE event_id = $1`, h.DbSchema)
		_, err := tx.Exec(ctx, deleteQuery, eventId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to delete existing urls: %v", err),
			}
		}

		// Insert new type-genre pairs
		insertQuery := fmt.Sprintf(`INSERT INTO %s.event_link (event_id, label, type, url) VALUES ($1, $2, $3, $4)`, h.DbSchema)

		for _, url := range req.Types {
			_, err = tx.Exec(ctx, insertQuery, eventId, url.Label, url.Type, url.Url)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to insert url"),
				}
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
