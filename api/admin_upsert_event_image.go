package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/pluto"
	"github.com/sndcds/uranus/model"
)

func (h *ApiHandler) AdminUpsertEventImage(gc *gin.Context) {
	userId := h.userId(gc)
	apiResponseType := "upsert-event-image"

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		JSONError(gc, apiResponseType, http.StatusBadRequest, "eventId is required")
		return
	}

	identifier := gc.Param("identifier")
	if !IsEventImageIdentifier(identifier) {
		JSONError(gc, apiResponseType, http.StatusBadRequest, "unknown identifier")
		return
	}

	context := "event"
	fileNamePrefix := "event"

	fmt.Println("AdminUpsertEventImage")
	fmt.Println("   context: ", context)
	fmt.Println("   eventId: ", eventId)
	fmt.Println("   identifier: ", identifier)
	fmt.Println("   fileNamePrefix: ", fileNamePrefix)
	fmt.Println("   userId: ", userId)

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

	data := model.UpsertImageResultData{
		HttpStatus:        plutoUpsertImageResult.HttpStatus,
		Message:           plutoUpsertImageResult.Message,
		FileReplaced:      plutoUpsertImageResult.FileRemovedFlag,
		CacheFilesRemoved: plutoUpsertImageResult.CacheFilesRemoved,
		ImageId:           plutoUpsertImageResult.ImageId,
		ImageIdentifier:   identifier,
	}

	JSONSuccess(gc, "upsert_image_result", data, nil)
}
