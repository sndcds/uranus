package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *ApiHandler) AdminDeleteOrganizer(gc *gin.Context) {
	pool := h.DbPool
	ctx := gc.Request.Context()

	organizerIdStr := gc.Param("organizerId")
	if organizerIdStr == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Organizer ID is required"})
		return
	}

	organizerId, err := strconv.Atoi(organizerIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organizer ID"})
		return
	}

	query := fmt.Sprintf(`DELETE FROM %s.organizer WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := pool.Exec(ctx, query, organizerId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete organizer", "details": err.Error()})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "Organizer not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Organizer deleted successfully", "id": organizerId})
}
