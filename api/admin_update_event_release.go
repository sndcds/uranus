package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminUpdateEventReleaseStatus(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool
	dbSchema := h.Config.DbSchema

	eventId := gc.Param("eventId")
	if eventId == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "event ID is required"})
		return
	}

	type Incoming struct {
		ReleaseDate     *string `json:"release_date"`
		ReleaseStatusId int     `json:"release_status_id" binding:"required"`
	}

	fmt.Println("... 1:")
	var req Incoming
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("... 2:")
	fmt.Println("ReleaseStatusId:", req.ReleaseStatusId)
	fmt.Println("ReleaseDate:", req.ReleaseDate)
	// var releaseDate interface{} = req.ReleaseDate
	// if releaseDate != nil && *req.ReleaseDate == "" {
	//	releaseDate = nil
	//}

	sqlTemplate := `
		UPDATE {{schema}}.event
		SET release_status_id = $2,
		    release_date = $3
		WHERE id = $1`
	sqlQuery := strings.Replace(sqlTemplate, "{{schema}}", dbSchema, 1)

	res, err := pool.Exec(ctx, sqlQuery, eventId, req.ReleaseStatusId, req.ReleaseDate)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update event status: %v", err)})
		return
	}

	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"event_id": eventId,
		"message":  "event release status updated successfully",
	})
}
