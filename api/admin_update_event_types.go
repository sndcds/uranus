package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminUpdateEventTypes(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-update-event-types")
	ctx := gc.Request.Context()

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "eventUuid is required")
		return
	}

	type eventTypesRequest struct {
		Types []struct {
			TypeId  int  `json:"type_id" binding:"required"`
			GenreId *int `json:"genre_id"`
		} `json:"event_types" binding:"required"`
	}

	var req eventTypesRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		debugf(err.Error())
		apiRequest.PayloadError()
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		// Delete existing type-genre links
		deleteQuery := fmt.Sprintf(`DELETE FROM %s.event_type_link WHERE event_uuid = $1::uuid`, h.DbSchema)
		debugf(deleteQuery)
		debugf(eventUuid)
		_, err := tx.Exec(ctx, deleteQuery, eventUuid)
		if err != nil {
			debugf(err.Error())
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("failed to delete existing type-genre links: %v", err),
			}
		}

		// Insert new type-genre pairs
		insertQuery := fmt.Sprintf(`INSERT INTO %s.event_type_link (event_uuid, type_id, genre_id) VALUES ($1::uuid, $2, $3)`, h.DbSchema)

		for _, pair := range req.Types {
			genreId := 0
			if pair.GenreId != nil {
				genreId = *pair.GenreId
			}
			_, err = tx.Exec(ctx, insertQuery, eventUuid, pair.TypeId, genreId)
			if err != nil {
				return &ApiTxError{
					Code: http.StatusInternalServerError,
					Err:  fmt.Errorf("failed to insert type_id=%d, genre_id=%d: %v", pair.TypeId, pair.GenreId, err),
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
		debugf(txErr.Error())
		apiRequest.DatabaseError()
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "")
}
