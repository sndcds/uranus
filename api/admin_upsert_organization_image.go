package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/pluto"
)

func (h *ApiHandler) AdminUpsertOrganizationImage(gc *gin.Context) {
	userId := h.userId(gc)

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

	context := "organization"
	fileNamePrefix := "org"

	// Upsert image in Pluto
	plutoUpsertImageResult, err := pluto.UpsertImage(
		gc, context,
		orgId,
		identifier,
		&fileNamePrefix,
		userId,
		RefreshEventProjectionsCallback("organization", []int{orgId}))
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
