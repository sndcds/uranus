package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteEvent(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	if !h.VerifyUserPassword(gc, userId) {
		return
	}

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event Id"})
		return
	}

	deleteSql := fmt.Sprintf(`DELETE FROM %s.event WHERE id = $1`, h.Config.DbSchema)

	cmdTag, err := h.DbPool.Exec(ctx, deleteSql, eventId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event", "details": err.Error()})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Event deleted successfully", "id": eventId})
}
