package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteEventMainImage(gc *gin.Context) {
	ctx := gc.Request.Context()

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	txErr := WithTransaction(ctx, h.DbPool, func(tx pgx.Tx) *ApiTxError {
		query := fmt.Sprintf(`
SELECT gen_file_name, pi.id FROM %s.event_image_link AS eil
JOIN %s.pluto_image pi ON pi.id = eil.pluto_image_id
WHERE eil.event_id = $1 AND eil.main_image = TRUE`,
			h.DbSchema, h.DbSchema)

		var plutoImageId int
		var plutoGenFileName string
		err := tx.QueryRow(ctx, query, eventId).Scan(&plutoGenFileName, &plutoImageId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Query failed: %v", err),
			}
		}

		query = fmt.Sprintf(`DELETE FROM %s.event_image_link WHERE event_id = $1 AND main_image = TRUE`, h.DbSchema)
		_, err = tx.Exec(ctx, query, eventId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Query failed: %v", err),
			}
		}

		query = fmt.Sprintf(`DELETE FROM %s.pluto_image WHERE id = $1`, h.DbSchema)
		_, err = tx.Exec(ctx, query, plutoImageId)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Query failed: %v", err),
			}
		}

		if len(plutoGenFileName) <= 0 {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("No Pluto image file to delete: %v", err),
			}
		}

		filePath := h.Config.PlutoImageDir + "/" + plutoGenFileName
		err = os.Remove(filePath)
		if err != nil {
			return &ApiTxError{
				Code: http.StatusInternalServerError,
				Err:  fmt.Errorf("Error removing file: %v", err),
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
		"message":  "event image deleted successfully",
		"event_id": eventId,
	})
}
