package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grainsapi"
)

func (h *ApiHandler) AdminDeleteEvent(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grainsapi.NewRequest(gc, "admin-delete-event")

	err := h.VerifyUserPassword(gc, userId)
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

	query := fmt.Sprintf(`DELETE FROM %s.event WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, eventId)
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
