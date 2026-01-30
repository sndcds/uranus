package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/pluto"
)

func (h *ApiHandler) AdminUpsertEventImage(gc *gin.Context) {
	userId := h.userId(gc)

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
		return
	}

	identifier := gc.Param("identifier")
	if !IsEventImageIdentifier(identifier) {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "unknown identifier"})
		return
	}

	context := "event"
	fileNamePrefix := "event"

	// Upsert image in Pluto
	plutoUpsertImageResult, err := pluto.UpsertImage(
		gc, context,
		eventId,
		identifier,
		&fileNamePrefix,
		userId,
		RefreshEventProjectionsCallback("event", []int{eventId}))
	if err != nil || plutoUpsertImageResult.HttpStatus != http.StatusOK {
		gc.JSON(plutoUpsertImageResult.HttpStatus, gin.H{"error": plutoUpsertImageResult.Message})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"http_status":         plutoUpsertImageResult.HttpStatus,
		"message":             plutoUpsertImageResult.Message,
		"file_replaced":       plutoUpsertImageResult.FileRemovedFlag,
		"cache_files_removed": plutoUpsertImageResult.CacheFilesRemoved,
		"image_id":            plutoUpsertImageResult.ImageId,
		"image_identifier":    identifier,
	})
}
