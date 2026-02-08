package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grainsapi"
)

func (h *ApiHandler) AdminDeleteSpace(gc *gin.Context) {
	ctx := gc.Request.Context()
	userId := h.userId(gc)
	apiRequest := grainsapi.NewRequest(gc, "admin-delete-space")

	err := h.VerifyUserPassword(gc, userId)
	if err != nil {
		apiRequest.Error(http.StatusUnauthorized, err.Error())
		return
	}

	spaceId, ok := ParamInt(gc, "spaceId")
	if !ok {
		apiRequest.Error(http.StatusBadRequest, "spaceId is required")
		return
	}
	apiRequest.SetMeta("space_id", spaceId)

	query := fmt.Sprintf(`DELETE FROM %s.space WHERE id = $1`, h.Config.DbSchema)
	cmdTag, err := h.DbPool.Exec(ctx, query, spaceId)
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
