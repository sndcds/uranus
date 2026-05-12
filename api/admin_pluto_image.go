package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/pluto"
)

func (h *ApiHandler) AdminDeletePlutoImage(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-pluto-image")

	plutoContext := gc.Param("context")

	contextUuid := gc.Param("contextUuid")
	if contextUuid == "" {
		apiRequest.Required("contextUuid is required")
		return
	}

	validator, err := validatorByContext(plutoContext)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	refresher, err := refresherByContext(plutoContext)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	identifier := gc.Param("identifier")
	if !validator(identifier) {
		apiRequest.Required("unknown identifier")
		return
	}

	plutoDeleteImageResult, err := pluto.DeleteImage(
		gc, plutoContext, contextUuid, identifier,
		refresher(plutoContext, []string{contextUuid}))
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

	plutoContext := gc.Param("context")
	apiRequest.SetMeta("pluto_context", plutoContext)

	contextUuid := gc.Param("contextUuid")
	if contextUuid == "" {
		apiRequest.Required("contextUuid is required")
		return
	}
	apiRequest.SetMeta("pluto_context_uuid", contextUuid)

	validator, err := validatorByContext(plutoContext)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	refresher, err := refresherByContext(plutoContext)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	identifier := gc.Param("identifier")
	apiRequest.SetMeta("pluto_image_identifier", identifier)
	if !validator(identifier) {
		apiRequest.Error(http.StatusBadRequest, "unknown identifier")
		return
	}

	fileNamePrefix, err := fileNamePrefixByContext(plutoContext)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	debugf("context: %s, contextUuid: %s, identifier: %s", plutoContext, contextUuid, identifier)

	// Upsert image in Pluto
	plutoUpsertImageResult, err := pluto.UpsertImage(
		gc,
		plutoContext,
		contextUuid,
		identifier,
		&fileNamePrefix,
		userUuid,
		refresher(plutoContext, []string{contextUuid}),
	)
	if err != nil || plutoUpsertImageResult.HttpStatus != http.StatusOK {
		debugf("err: %v", err)
		apiRequest.Error(plutoUpsertImageResult.HttpStatus, plutoUpsertImageResult.Message)
		return
	}

	apiRequest.SetMeta("file_replaced", plutoUpsertImageResult.FileRemovedFlag)
	apiRequest.SetMeta("cache_removed_count", plutoUpsertImageResult.CacheFilesRemoved)
	apiRequest.SetMeta("image_uuid", plutoUpsertImageResult.ImageUuid)
	apiRequest.SuccessNoData(plutoUpsertImageResult.HttpStatus, plutoUpsertImageResult.Message)
}

func (h *ApiHandler) AdminCleanupImages(gc *gin.Context) {
	err := pluto.CleanupImages(gc.Request.Context())
	if err != nil {
		gc.JSON(500, gin.H{"error": err.Error()})
		return
	}

	gc.JSON(200, gin.H{"status": "cleanup done"})
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
	case "portal":
		return IsPortalImageIdentifier, nil
	default:
		return nil, fmt.Errorf("unknown context: %s", context)
	}
}

func NoOpRefreshEventProjectionsCallback(entity string, uuids []string) pluto.TxFunc {
	return func(ctx context.Context, tx pgx.Tx) error {
		return nil
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
	case "portal":
		return NoOpRefreshEventProjectionsCallback, nil
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
	case "portal":
		return "portal", nil
	default:
		return "", fmt.Errorf("unknown context: %s", context)
	}
}
