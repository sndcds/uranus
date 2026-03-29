package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminDeleteEvent(gc *gin.Context) {
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-delete-event")

	err := h.VerifyUserPassword(gc, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	eventId, ok := ParamInt(gc, "eventId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "eventId is required")
		return
	}
	apiRequest.SetMeta("event_id", eventId)

	cmdTag, err := h.DbPool.Exec(ctx, app.UranusInstance.SqlAdminGetEvent, eventId)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to delete event")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "event not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "event deleted successfully")
}
