package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminDeleteEventDate(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grains_api.NewRequest(gc, "admin-delete-event-date")

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

	eventDateId, ok := ParamInt(gc, "dateId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "dateId is required")
		return
	}
	apiRequest.SetMeta("event_date_id", eventDateId)

	query := fmt.Sprintf(`DELETE FROM %s.event_date WHERE id = $1`, h.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, eventDateId)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to delete event date")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "event date not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "event date deleted successfully")
}
