package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminDeleteVenue(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-venue")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	err := h.VerifyUserPassword(gc, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	venueUuid := gc.Param("venueUuid")
	if venueUuid == "" {
		apiRequest.Error(http.StatusBadRequest, "venueUuid is required")
		return
	}
	apiRequest.SetMeta("venue_uuid", venueUuid)

	query := fmt.Sprintf(`DELETE FROM %s.venue WHERE uuid = $1::uuid`, h.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, venueUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "venue not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "venue deleted successfully")
}
