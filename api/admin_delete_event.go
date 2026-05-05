package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminDeleteEvent(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-event")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	err := h.VerifyUserPassword(gc, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Required("eventUuid is required")
		return
	}
	apiRequest.SetMeta("event_uuid", eventUuid)

	cmdTag, err := h.DbPool.Exec(ctx, app.UranusInstance.SqlAdminDeleteEvent, eventUuid)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.NotFound("event not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "event deleted successfully")
}
