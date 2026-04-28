package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminUpdateEventLinks(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-event-links")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "eventUuid is required")
		return
	}

	type Payload struct {
		Types []struct {
			Label **string `json:"label"`
			Type  string   `json:"type" binding:"required"`
			Url   string   `json:"url" binding:"required"`
		} `json:"event_links" binding:"required"`
		SourceLink *string `json:"source_link"`
	}

	var payload Payload
	if err := gc.ShouldBindJSON(&payload); err != nil {
		apiRequest.PayloadError()
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		permissions, err := h.GetUserEventOrganizerPermissionsTx(gc, tx, userUuid, eventUuid)
		if err != nil {
			return TxInternalError(nil)
		}
		if !permissions.Has(app.PermEditEvent) {
			return &ApiTxError{
				Code: http.StatusForbidden,
				Err:  errors.New("(#1) event not found"),
			}
		}

		// Update source link
		query := fmt.Sprintf(`UPDATE %s.event SET source_link = $1 WHERE uuid = $2::uuid`, h.DbSchema)
		_, err = tx.Exec(ctx, query, payload.SourceLink, eventUuid)
		if err != nil {
			return TxInternalError(nil)
		}

		// Delete existing type-genre links
		deleteQuery := fmt.Sprintf(`DELETE FROM %s.event_link WHERE event_uuid = $1::uuid`, h.DbSchema)
		_, err = tx.Exec(ctx, deleteQuery, eventUuid)
		if err != nil {
			return TxInternalError(nil)
		}

		// Insert new type-genre pairs
		insertQuery := fmt.Sprintf(`INSERT INTO %s.event_link (event_uuid, label, type, url) VALUES ($1::uuid, $2, $3, $4)`, h.DbSchema)
		for _, url := range payload.Types {
			_, err = tx.Exec(ctx, insertQuery, eventUuid, url.Label, url.Type, url.Url)
			if err != nil {
				return TxInternalError(nil)
			}
		}

		err = RefreshEventProjections(ctx, tx, "event", []string{eventUuid})
		if err != nil {
			return TxInternalError(nil)
		}

		return nil
	})
	if txErr != nil {
		apiRequest.Error(txErr.Code, txErr.Error())
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}
