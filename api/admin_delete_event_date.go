package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminDeleteEventDate(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	if !h.VerifyUserPassword(gc, userId) {
		return // Already sent JSON error
	}

	eventId, ok := ParamInt(gc, "eventId")
	fmt.Println("eventId", eventId)
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
		return
	}

	eventDateId, ok := ParamInt(gc, "dateId")
	fmt.Println("eventDateId", eventDateId)
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "dateId is required"})
		return
	}

	query := fmt.Sprintf(`DELETE FROM %s.event_date WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, eventDateId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event date", "details": err.Error()})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "Event date not found"})
		return
	}

	gc.JSON(
		http.StatusOK,
		gin.H{
			"message":       "Event date deleted successfully",
			"event_id":      eventId,
			"event_date_id": eventDateId,
		})
}
