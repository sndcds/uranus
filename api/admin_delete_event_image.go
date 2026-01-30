package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/pluto"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteEventImage(gc *gin.Context) {
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

	plutoDeleteImageResult, err := pluto.DeleteImage(
		gc,
		"event",
		eventId,
		identifier,
		RefreshEventProjectionsCallback("event", []int{eventId}))
	if err != nil {
		gc.JSON(plutoDeleteImageResult.HttpStatus, gin.H{"error": plutoDeleteImageResult.Message})
		return
	}

	gc.JSON(http.StatusOK, gin.H{
		"http_status":         plutoDeleteImageResult.HttpStatus,
		"message":             plutoDeleteImageResult.Message,
		"file_removed":        plutoDeleteImageResult.FileRemovedFlag,
		"cache_files_removed": plutoDeleteImageResult.CacheFilesRemoved,
		"image_id":            plutoDeleteImageResult.ImageId,
		"image_identifier":    identifier,
	})
}
