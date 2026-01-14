package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/uranus/app"
)

// TODO: Code review

func (h *ApiHandler) AdminGetPermissionsList(gc *gin.Context) {
	ctx := gc.Request.Context()
	lang := gc.DefaultQuery("lang", "en")

	var permissionsJSON []byte

	err := h.DbPool.QueryRow(ctx, app.UranusInstance.SqlAdminGetPermissionList, lang).Scan(&permissionsJSON)
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
