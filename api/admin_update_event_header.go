package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

// Payload for updating title, subtitle, and description
type UpdateEventRequest struct {
	Title    string  `json:"title" binding:"required"`
	Subtitle *string `json:"subtitle,omitempty"`
}

func (h *ApiHandler) AdminUpdateEventHeader(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	// Get event Id from URL
	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event Id is required"})
		return
	}

	var req UpdateEventRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build query dynamically depending on whether subtitle is provided
	var sqlQuery string
	var args []interface{}

	if req.Subtitle != nil {
		sqlTemplate := `UPDATE {{schema}}.event SET title = $2, subtitle = $3 WHERE id = $1`
		sqlQuery = strings.Replace(sqlTemplate, "{{schema}}", dbSchema, 1)
		args = []interface{}{eventId, req.Title, req.Subtitle}
	} else {
		sqlTemplate := `UPDATE {{schema}}.event SET title = $2 WHERE id = $1`
		sqlQuery = strings.Replace(sqlTemplate, "{{schema}}", dbSchema, 1)
		args = []interface{}{eventId, req.Title}
	}

	// Execute update
	res, err := pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update event: %v", err)})
		return
	}

	if res.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventId,
		"message":  "event updated successfully",
	})
}
