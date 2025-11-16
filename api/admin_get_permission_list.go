package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetPermissionList(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	langStr := gc.DefaultQuery("lang", "en")

	var permissionsJSON []byte
	err := pool.QueryRow(ctx, app.Singleton.SqlAdminGetPermissionList, langStr).Scan(&permissionsJSON)
	if err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gc.Data(http.StatusOK, "application/json", permissionsJSON)
}
