package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/pluto"
)

func (h *ApiHandler) AdminDeletePlutoImage(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-pluto-image")

	context := gc.Param("context")
	debugf("context: %s", context)
	apiRequest.SetMeta("pluto_context", context)

	contextId, ok := ParamInt(gc, "contextId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "contextId is required")
		return
	}
	debugf("contextId: %d", contextId)
	apiRequest.SetMeta("pluto_context_id", contextId)

	validator, err := validatorByContext(context)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	refresher, err := refresherByContext(context)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	identifier := gc.Param("identifier")
	debugf("identifier: %s", identifier)
	apiRequest.SetMeta("pluto_image_identifier", identifier)
	if !validator(identifier) {
		apiRequest.Error(http.StatusBadRequest, "unknown identifier")
		return
	}

	plutoDeleteImageResult, err := pluto.DeleteImage(
		gc, context, contextId, identifier,
		refresher(context, []int{contextId}))
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "pluto api failed")
		return
	}

	apiRequest.SetMeta("file_removed", plutoDeleteImageResult.FileRemovedFlag)
	apiRequest.SetMeta("cache_removed_count", plutoDeleteImageResult.CacheFilesRemoved)
	apiRequest.SetMeta("image_id", plutoDeleteImageResult.ImageId)
	apiRequest.SuccessNoData(plutoDeleteImageResult.HttpStatus, plutoDeleteImageResult.Message)
}

func (h *ApiHandler) AdminUpsertPlutoImage(gc *gin.Context) {
	userId := h.userId(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-upsert-pluto-image")

	context := gc.Param("context")
	debugf("context: %s", context)
	apiRequest.SetMeta("pluto_context", context)

	contextId, ok := ParamInt(gc, "contextId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "contextId is required")
		return
	}
	debugf("contextId: %s", contextId)
	apiRequest.SetMeta("pluto_context_id", contextId)

	validator, err := validatorByContext(context)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	refresher, err := refresherByContext(context)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	identifier := gc.Param("identifier")
	debugf("identifier: %s", identifier)
	apiRequest.SetMeta("pluto_image_identifier", identifier)
	if !validator(identifier) {
		apiRequest.Error(http.StatusBadRequest, "unknown identifier")
		return
	}

	fileNamePrefix, err := fileNamePrefixByContext(context)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	// Upsert image in Pluto
	plutoUpsertImageResult, err := pluto.UpsertImage(
		gc, context, contextId, identifier, &fileNamePrefix, userId,
		refresher(context, []int{contextId}))
	if err != nil || plutoUpsertImageResult.HttpStatus != http.StatusOK {
		gc.JSON(plutoUpsertImageResult.HttpStatus, gin.H{"error": plutoUpsertImageResult.Message})
		return
	}

	apiRequest.SetMeta("file_replaced", plutoUpsertImageResult.FileRemovedFlag)
	apiRequest.SetMeta("cache_removed_count", plutoUpsertImageResult.CacheFilesRemoved)
	apiRequest.SetMeta("image_id", plutoUpsertImageResult.ImageId)
	apiRequest.SuccessNoData(plutoUpsertImageResult.HttpStatus, plutoUpsertImageResult.Message)
}

func validatorByContext(context string) (pluto.ImageIdentifierValidator, error) {
	switch context {
	case "organization":
		return IsOrganizationImageIdentifier, nil
	case "venue":
		return IsVenueImageIdentifier, nil
	/*
		case "space":
			validator = IsSpaceImageIdentifier
	*/
	case "event":
		return IsEventImageIdentifier, nil
	default:
		return nil, fmt.Errorf("unknown context: %s", context)
	}
}

func refresherByContext(context string) (pluto.ImageRefresherCallback, error) {
	switch context {
	case "organization":
		return RefreshEventProjectionsCallback, nil
	case "venue":
		return RefreshEventProjectionsCallback, nil
	/*
		case "space":
			validator = IsSpaceImageIdentifier
	*/
	case "event":
		return RefreshEventProjectionsCallback, nil
	default:
		return nil, fmt.Errorf("unknown context: %s", context)
	}
}

func fileNamePrefixByContext(context string) (string, error) {
	switch context {
	case "organization":
		return "org", nil
	case "venue":
		return "venue", nil
	/*
		case "space":
			validator = IsSpaceImageIdentifier
	*/
	case "event":
		return "event", nil
	default:
		return "", fmt.Errorf("unknown context: %s", context)
	}
}
