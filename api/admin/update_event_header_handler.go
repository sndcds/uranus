package api_admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

// Payload for updating title, subtitle, and description
type UpdateEventRequest struct {
	Title       string  `json:"title" binding:"required"`
	Subtitle    *string `json:"subtitle,omitempty"`
	Description string  `json:"description" binding:"required"`
}

func UpdateEventHeaderHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool
	dbSchema := app.Singleton.Config.DbSchema

	// Get event ID from URL
	eventID := gc.Param("id")
	if eventID == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
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
		sqlTemplate := `
			UPDATE {{schema}}.event
			SET title = $1,
			    subtitle = $2,
			    description = $3
			WHERE id = $4`
		sqlQuery = strings.Replace(sqlTemplate, "{{schema}}", dbSchema, 1)
		args = []interface{}{req.Title, req.Subtitle, req.Description, eventID}
	} else {
		sqlTemplate := `
			UPDATE {{schema}}.event
			SET title = $1,
			    description = $2
			WHERE id = $3`
		sqlQuery = strings.Replace(sqlTemplate, "{{schema}}", dbSchema, 1)
		args = []interface{}{req.Title, req.Description, eventID}
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
		"event_id": eventID,
		"message":  "event updated successfully",
	})
}
