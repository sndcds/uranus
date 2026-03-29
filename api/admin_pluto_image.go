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

	contextUuid := gc.Param("contextUuid")
	if contextUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "contextUuid is required")
		return
	}

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
	if !validator(identifier) {
		apiRequest.Error(http.StatusBadRequest, "unknown identifier")
		return
	}

	plutoDeleteImageResult, err := pluto.DeleteImage(
		gc, context, contextUuid, identifier,
		refresher(context, []string{contextUuid}))
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "pluto api failed")
		return
	}

	apiRequest.SetMeta("file_removed", plutoDeleteImageResult.FileRemovedFlag)
	apiRequest.SetMeta("cache_removed_count", plutoDeleteImageResult.CacheFilesRemoved)
	apiRequest.SetMeta("image_uuid", plutoDeleteImageResult.ImageUuid)
	apiRequest.SuccessNoData(plutoDeleteImageResult.HttpStatus, plutoDeleteImageResult.Message)
}

func (h *ApiHandler) AdminUpsertPlutoImage(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-upsert-pluto-image")
	userUuid := h.userUuid(gc)

	// TODO: Check user permissions for different contexts

	context := gc.Param("context")
	apiRequest.SetMeta("pluto_context", context)

	contextUuid := gc.Param("contextUuid")
	debugf("contextUuid: %s", contextUuid)
	if contextUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "contextUuid is required")
		return
	}
	apiRequest.SetMeta("pluto_context_uuid", contextUuid)

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
	apiRequest.SetMeta("pluto_image_identifier", identifier)
	debugf("identifier: %s", identifier)
	if !validator(identifier) {
		apiRequest.Error(http.StatusBadRequest, "unknown identifier")
		return
	}

	fileNamePrefix, err := fileNamePrefixByContext(context)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}
	debugf("fileNamePrefix: %s", fileNamePrefix)

	// Upsert image in Pluto
	plutoUpsertImageResult, err := pluto.UpsertImage(
		gc, context, contextUuid, identifier, &fileNamePrefix, userUuid,
		refresher(context, []string{contextUuid}))
	if err != nil || plutoUpsertImageResult.HttpStatus != http.StatusOK {
		debugf("err: %v", err)
		gc.JSON(plutoUpsertImageResult.HttpStatus, gin.H{"error": plutoUpsertImageResult.Message})
		return
	}

	debugf("pluto.UpsertImage ok")

	apiRequest.SetMeta("file_replaced", plutoUpsertImageResult.FileRemovedFlag)
	apiRequest.SetMeta("cache_removed_count", plutoUpsertImageResult.CacheFilesRemoved)
	apiRequest.SetMeta("image_id", plutoUpsertImageResult.ImageUuid)
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
