package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type eventTypesRequest struct {
	Types []struct {
		TypeId  int  `json:"type_id" binding:"required"`
		GenreId *int `json:"genre_id"`
	} `json:"types" binding:"required"`
}

func (h *ApiHandler) AdminUpdateEventTypes(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	var req eventTypesRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Delete existing type-genre links
		deleteQuery := fmt.Sprintf(`DELETE FROM %s.event_type_link WHERE event_id = $1`, h.DbSchema)
		_, err := tx.Exec(ctx, deleteQuery, eventId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to delete existing type-genre links: %v", err),
			}
		}

		// Insert new type-genre pairs
		insertQuery := fmt.Sprintf(`INSERT INTO %s.event_type_link (event_id, type_id, genre_id) VALUES ($1, $2, $3)`, h.DbSchema)

		for _, pair := range req.Types {
			genreId := 0
			if pair.GenreId != nil {
				genreId = *pair.GenreId
			}
			_, err = tx.Exec(ctx, insertQuery, eventId, pair.TypeId, genreId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to insert type_id=%d, genre_id=%d: %v", pair.TypeId, pair.GenreId, err),
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
		gc.JSON(txErr.Code, gin.H{"error": txErr.Error()})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"message":  "event types and genres updated successfully",
		"event_id": eventId,
	})
}
