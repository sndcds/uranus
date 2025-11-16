package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetPermissionList(gc *gin.Context) {
	ctx := gc.Request.Context()
	pool := h.DbPool

	langStr := gc.DefaultQuery("lang", "en")

	var permissionsJSON []byte

	err := pool.QueryRow(ctx, app.Singleton.SqlAdminGetPermissionList, langStr).Scan(&permissionsJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			permissionsJSON = []byte("{}")
		} else {
			gc.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	gc.Data(http.StatusOK, "application/json", permissionsJSON)
}
