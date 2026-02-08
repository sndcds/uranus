package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grainsapi"
)

func (h *ApiHandler) AdminDeleteVenue(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grainsapi.NewRequest(gc, "admin-delete-venue")

	err := h.VerifyUserPassword(gc, userId)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	venueId, ok := ParamInt(gc, "venueId")
	if !ok {
		apiRequest.Error(http.StatusUnauthorized, "venueId is required")
		return
	}
	apiRequest.SetMeta("venueId", venueId)

	query := fmt.Sprintf(`DELETE FROM %s.venue WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, venueId)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to delete venue")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "venue not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "venue deleted successfully")
}
