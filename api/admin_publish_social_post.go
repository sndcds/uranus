package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
	"github.com/sndcds/uranus/service"
)

func (h *ApiHandler) AdminPublishSocialPost(gc *gin.Context) {
	request := grains_api.NewRequest(gc, "admin-publish-social-post")
	if !app.ValidUUID(h.userUuid(gc)) {
		request.Error(http.StatusUnauthorized, "authentication required")
		return
	}
	postUUID, ok := socialPostUUID(gc, request)
	if !ok {
		return
	}
	result := model.SocialPostPublish{PostUuid: postUUID, Results: []model.SocialTargetPublish{}}
	work, txErr := h.claimSocialPublish(gc, postUUID, &result)
	if txErr != nil {
		socialPostRespondError(request, txErr)
		return
	}
	status, attempted := http.StatusOK, false
	for i, task := range work {
		if !task.claimed {
			if status != http.StatusInternalServerError {
				status = http.StatusMultiStatus
			}
			continue
		}
		attempted = true
		target := &result.Results[i]
		published, err := h.publishSocialTarget(gc.Request.Context(), task)
		target.Status = "published"
		if err == nil {
			target.RemotePostID = &published.RemotePostID
		} else {
			target.Status = "failed"
			if service.SocialPublishUncertain(err) {
				target.Status = "publishing"
			}
			message := err.Error()
			target.Error = &message
			if status != http.StatusInternalServerError {
				status = http.StatusMultiStatus
			}
		}
		if !h.finishSocialPublish(gc.Request.Context(), *target) {
			// Preserve earlier targets; a failed local write cannot undo a remote
			// publication. The persisted claim remains blocked for reconciliation.
			target.Status = "publishing"
			target.RemotePostID = nil
			message := "publication outcome could not be stored; manual reconciliation required"
			target.Error = &message
			status = http.StatusInternalServerError
		}
	}
	if !attempted {
		status = http.StatusConflict
	}
	request.Success(status, result)
}
