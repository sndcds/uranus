package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

type UpdateEventTeaserRequest struct {
	TeaserText string `json:"teaser_text" binding:"required"`
}

func (h *ApiHandler) AdminUpdateEventTeaser(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	// Get event ID from URL
	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	var req UpdateEventTeaserRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sqlTemplate := `
		UPDATE {{schema}}.event
		SET teaser_text = $2
		WHERE id = $1`
	sqlQuery := strings.Replace(sqlTemplate, "{{schema}}", dbSchema, 1)

	// Execute update
	res, err := pool.Exec(ctx, sqlQuery, eventId, req.TeaserText)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update event: %v", err)})
		return
	}

	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventId,
		"message":  "event description updated successfully",
	})
}
