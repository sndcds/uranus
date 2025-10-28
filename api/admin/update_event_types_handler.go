package api_admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type UpdateEventTypesRequest struct {
	Types []struct {
		TypeId  int  `json:"type_id" binding:"required"`
		GenreId *int `json:"genre_id"`
	} `json:"types" binding:"required"`
}

func UpdateEventTypesHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool
	dbSchema := app.Singleton.Config.DbSchema

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	var req UpdateEventTypesRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Delete existing type-genre links
	sqlDelete := strings.Replace(`DELETE FROM {{schema}}.event_type_links WHERE event_id = $1`, "{{schema}}", dbSchema, 1)
	if _, err = tx.Exec(ctx, sqlDelete, eventId); err != nil {
		_ = tx.Rollback(ctx)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete existing type-genre links: %v", err)})
		return
	}

	// Insert new type-genre pairs
	sqlInsert := strings.Replace(
		`INSERT INTO {{schema}}.event_type_links (event_id, type_id, genre_id) VALUES ($1, $2, $3)`,
		"{{schema}}", dbSchema, 1,
	)

	for _, pair := range req.Types {
		if _, err = tx.Exec(ctx, sqlInsert, eventId, pair.TypeId, pair.GenreId); err != nil {
			_ = tx.Rollback(ctx)
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert type_id=%d, genre_id=%d: %v", pair.TypeId, pair.GenreId, err)})
			return
		}
	}

	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventId,
		"message":  "event types and genres updated successfully",
	})
}
