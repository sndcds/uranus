package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
)

func (h *ApiHandler) AdminDeleteSpace(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-delete-space")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	err := h.VerifyUserPassword(gc, userUuid)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	spaceUuid := gc.Param("spaceUuid")
	if spaceUuid == "" {
		apiRequest.Required("spaceUuid is required")
		return
	}
	apiRequest.SetMeta("space_uuid", spaceUuid)

	query := fmt.Sprintf(`DELETE FROM %s.space WHERE uuid = $1::uuid`, h.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, spaceUuid)
	if err != nil {
		apiRequest.Error(http.StatusInternalServerError, "failed to delete space")
		return
	}

	if cmdTag.RowsAffected() == 0 {
		apiRequest.Error(http.StatusNotFound, "space not found")
		return
	}

	apiRequest.SuccessNoData(http.StatusOK, "space deleted successfully")
}
