package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminDeleteEventDate(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := gc.GetInt("user-id")

	fmt.Println("userId:", userId)

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
		return
	}
	fmt.Println("eventId:", eventId)

	eventDateId, ok := ParamInt(gc, "dateId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "dateId is required"})
		return
	}
	fmt.Println("eventDateId:", eventDateId)

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
