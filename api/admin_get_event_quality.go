package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/service"
)

func (h *ApiHandler) AdminGetEventQuality(gc *gin.Context) {
	req := grains_api.NewRequest(gc, "admin-get-event-quality")
	eventUuid := gc.Param("eventUuid")
	var id pgtype.UUID
	if err := id.Scan(eventUuid); err != nil || !id.Valid {
		req.Error(http.StatusBadRequest, "Parameter eventUuid must be a valid UUID")
		return
	}

	event, err := h.loadAdminEvent(gc.Request.Context(), eventUuid, "de", h.userUuid(gc))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			req.Error(http.StatusNotFound, "Event not found")
			return
		}
		debugf("%v", err)
		req.InternalServerError()
		return
	}
	req.SetMeta("language", "de")
	req.Success(http.StatusOK, service.EvaluateEventQuality(event))
}
