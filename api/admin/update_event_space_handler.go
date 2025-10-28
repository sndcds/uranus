package api_admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

type updateEventSpaceRequest struct {
	SpaceId int `json:"space_id" binding:"required"`
}

func UpdateEventSpaceHandler(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := app.Singleton.MainDbPool
	dbSchema := app.Singleton.Config.DbSchema

	// Get event ID from URL
	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	var req updateEventSpaceRequest
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sqlTemplate := `UPDATE {{schema}}.event SET space_id = $2 WHERE id = $1`
	sqlQuery := strings.Replace(sqlTemplate, "{{schema}}", dbSchema, 1)

	res, err := pool.Exec(ctx, sqlQuery, eventId, req.SpaceId)
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
		"message":  "event space updated successfully",
	})
}
