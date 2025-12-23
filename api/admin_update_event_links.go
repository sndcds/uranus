package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type upsertEventLinkRequest struct {
	Url     string `json:"url" binding:"required"`
	Title   string `json:"title"`
	UrlType int    `json:"url_type"`
}

func (h *ApiHandler) AdminUpsertEventLink(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	linkId, _ := ParamInt(gc, "linkId") // if not present, will be treated as 0

	var req upsertEventLinkRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		if linkId == 0 {
			// Insert new link
			insertQuery := fmt.Sprintf(
				`INSERT INTO %s.event_url (event_id, url, title, url_type) VALUES ($1, $2, $3, $4) RETURNING id`,
				h.DbSchema,
			)
			var newId int
			err := tx.QueryRow(ctx, insertQuery, eventId, req.Url, req.Title, req.UrlType).Scan(&newId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to insert event link: %v", err),
				}
			}
			linkId = newId // assign generated ID for response
		} else {
			// Update existing link
			updateQuery := fmt.Sprintf(
				`UPDATE %s.event_url SET url=$1, title=$2, url_type=$3 WHERE id=$4 AND event_id=$5`,
				h.DbSchema,
			)
			cmdTag, err := tx.Exec(ctx, updateQuery, req.Url, req.Title, req.UrlType, linkId, eventId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to update event link: %v", err),
				}
			}
			if cmdTag.RowsAffected() == 0 {
				return &ApiTxError{
					Code: http.StatusNotFound,
					Err:  fmt.Errorf("event link not found or does not belong to event"),
				}
			}
		}

		return nil
	})
	if txErr != nil {
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":  "event link saved successfully",
		"event_id": eventId,
		"link_id":  linkId,
	})
}
