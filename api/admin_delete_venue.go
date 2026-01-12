package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)

	if !h.VerifyUserPassword(gc, userId) {
		return
	}

	venueIdStr := gc.Param("venueId")
	if venueIdStr == "" {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Venue Id is required"})
		return
	}

	venueId, err := strconv.Atoi(venueIdStr)
	if err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "Invalid venue Id"})
		return
	}

	query := fmt.Sprintf(`DELETE FROM %s.venue WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, venueId)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete venue", "details": err.Error()})
		return
	}

	if cmdTag.RowsAffected() == 0 {
		gc.JSON(http.StatusNotFound, gin.H{"error": "Venue not found"})
		return
	}

	gc.JSON(http.StatusOK, gin.H{"message": "Venue deleted successfully", "id": venueId})
}
