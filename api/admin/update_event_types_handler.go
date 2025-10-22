package api_admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// Payload to update event types and genres
type UpdateEventTypesRequest struct {
	EventTypeIds []int `json:"event_type_ids" binding:"required"`
	GenreTypeIds []int `json:"genre_type_ids" binding:"required"`
}

func UpdateEventTypesHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool
	dbSchema := app.Singleton.Config.DbSchema

	// Get event ID from URL
	eventID := gc.Param("id")
	if eventID == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	var req UpdateEventTypesRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Begin transaction
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

	//  Delete existing event type links
	sqlDeleteTypes := strings.Replace(`DELETE FROM {{schema}}.event_type_links WHERE event_id = $1`, "{{schema}}", dbSchema, 1)
	if _, err = tx.Exec(ctx, sqlDeleteTypes, eventID); err != nil {
		_ = tx.Rollback(ctx)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete existing event types: %v", err)})
		return
	}

	// Insert new event type links
	sqlInsertType := strings.Replace(`INSERT INTO {{schema}}.event_type_links (event_id, type_id) VALUES ($1, $2)`, "{{schema}}", dbSchema, 1)
	for _, typeID := range req.EventTypeIds {
		if _, err = tx.Exec(ctx, sqlInsertType, eventID, typeID); err != nil {
			_ = tx.Rollback(ctx)
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert event type %d: %v", typeID, err)})
			return
		}
	}

	// Delete existing genre links
	sqlDeleteGenres := strings.Replace(`DELETE FROM {{schema}}.event_genre_links WHERE event_id = $1`, "{{schema}}", dbSchema, 1)
	if _, err = tx.Exec(ctx, sqlDeleteGenres, eventID); err != nil {
		_ = tx.Rollback(ctx)
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete existing genres: %v", err)})
		return
	}

	// Insert new genre links
	sqlInsertGenre := strings.Replace(`INSERT INTO {{schema}}.event_genre_links (event_id, type_id) VALUES ($1, $2)`, "{{schema}}", dbSchema, 1)
	for _, genreID := range req.GenreTypeIds {
		if _, err = tx.Exec(ctx, sqlInsertGenre, eventID, genreID); err != nil {
			_ = tx.Rollback(ctx)
			gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to insert genre %d: %v", genreID, err)})
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to commit transaction: %v", err)})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventID,
		"message":  "event types and genres updated successfully",
	})
}
