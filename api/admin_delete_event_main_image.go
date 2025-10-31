package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminDeleteEventMainImage(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool
	dbSchema := app.Singleton.Config.DbSchema

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := fmt.Sprintf(
		`SELECT gen_file_name, pi.id FROM %s.event_image_links AS eil
		JOIN %s.pluto_image pi ON pi.id = eil.pluto_image_id
		WHERE eil.event_id = $1 AND eil.main_image = TRUE`,
		dbSchema, dbSchema)

	var plutoImageId int
	var plutoGenFileName string
	err = tx.QueryRow(ctx, query, eventId).Scan(&plutoGenFileName, &plutoImageId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	query = fmt.Sprintf(`DELETE FROM %s.event_image_links WHERE event_id = $1 AND main_image = TRUE`, dbSchema)
	_, err = tx.Exec(ctx, query, eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	query = fmt.Sprintf(`DELETE FROM %s.pluto_image WHERE id = $1`, dbSchema)
	_, err = tx.Exec(ctx, query, plutoImageId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Query failed: %v", err)})
		return
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to commit transaction: %v", err)})
		return
	}

	if len(plutoGenFileName) <= 0 {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("No Pluto image file to delete: %v", err)})
		return
	}

	filePath := app.Singleton.Config.PlutoImageDir + "/" + plutoGenFileName
	fmt.Println("filePath:", filePath)

	err = os.Remove(filePath)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error removing file: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventId,
		"message":  "event image deleted successfully",
	})
}
