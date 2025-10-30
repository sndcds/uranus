package api_admin

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func UpdateEventMainImageHandler(gc *gin.Context) {
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
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	query := fmt.Sprintf(
		`SELECT gen_file_name FROM %s.event_image_links AS eil
		JOIN %s.pluto_image pi ON pi.id = eil.pluto_image_id
		WHERE eil.event_id = 422 AND eil.main_image = TRUE;`,
		dbSchema, dbSchema)

	var plutoGenFileName string
	err = tx.QueryRow(ctx, query).Scan(&plutoGenFileName)
	if err != nil {
		fmt.Println(err.Error())
		if err == sql.ErrNoRows {
			fmt.Println("No main image for this event")
		} else {
			fmt.Errorf("failed to query pluto_image_id: %w", err)
			return
		}
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
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
