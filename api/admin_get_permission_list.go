package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

func (h *ApiHandler) AdminGetPermissionsList(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-permissions-list")
	ctx := gc.Request.Context()
	lang := gc.DefaultQuery("lang", "en")

	var permissionsJSON []byte

	err := h.DbPool.QueryRow(ctx, app.UranusInstance.SqlAdminGetPermissionList, lang).Scan(&permissionsJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apiRequest.Success(http.StatusOK, gin.H{}, "")
			return
		}
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	var result any
	if err := json.Unmarshal(permissionsJSON, &result); err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, result, "")
}
