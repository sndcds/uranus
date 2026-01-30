package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/pluto"
)

// TODO: Review code

func (h *ApiHandler) AdminDeleteOrganizationImage(gc *gin.Context) {
	orgId, ok := ParamInt(gc, "orgId")
	if !ok {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "orgId is required"})
		return
	}

	identifier := gc.Param("identifier")
	if !IsOrganizationImageIdentifier(identifier) {
		gc.JSON(http.StatusBadRequest, gin.H{"error": "unknown identifier"})
		return
	}

	plutoDeleteImageResult, err := pluto.DeleteImage(
		gc,
		"organization",
		orgId,
		identifier,
		RefreshEventProjectionsCallback("organization", []int{orgId}))
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
