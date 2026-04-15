package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminDeleteEventDate(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-event-date")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	err := h.VerifyUserPassword(gc, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	eventUuid := gc.Param("eventUuid")
	if eventUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "eventUuid is required")
		return
	}
	apiRequest.SetMeta("event_uuid", eventUuid)

	eventDateUuid := gc.Param("dateUuid")
	if eventDateUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "dateUuid is required")
		return
	}
	apiRequest.SetMeta("event_date_uuid", eventDateUuid)

	query := fmt.Sprintf(`DELETE FROM %s.event_date WHERE uuid = $1::uuid`, h.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, eventDateUuid)
	if err != nil {
		apiRequest.InternalServerError()
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "event date not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "event date deleted successfully")
}
