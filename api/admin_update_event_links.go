package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminUpdateEventLinks(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-event-links")
	ctx := gc.Request.Context()

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "eventUuid is required")
		return
	}

	type eventLinksRequest struct {
		Types []struct {
			Label *string `json:"label"`
			Type  *string `json:"type"`
			Url   string  `json:"url" binding:"required"`
		} `json:"event_links" binding:"required"`
	}

	var req eventLinksRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		debugf(err.Error())
		apiRequest.PayloadError()
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Delete existing type-genre links
		deleteQuery := fmt.Sprintf(`DELETE FROM %s.event_link WHERE event_uuid = $1::uuid`, h.DbSchema)
		_, err := tx.Exec(ctx, deleteQuery, eventUuid)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to delete existing links: %v", err),
			}
		}

		// Insert new type-genre pairs
		insertQuery := fmt.Sprintf(`INSERT INTO %s.event_link (event_uuid, label, type, url) VALUES ($1::uuid, $2, $3, $4)`, h.DbSchema)

		for _, url := range req.Types {
			_, err = tx.Exec(ctx, insertQuery, eventUuid, url.Label, url.Type, url.Url)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  errors.New("failed to insert url"),
				}
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
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}
