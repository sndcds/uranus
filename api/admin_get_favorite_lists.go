package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

// PermissionNote: User must be authenticated.
// PermissionChecks: Enforced in SQL; no additional checks needed in Go.
// Verified: 2026-05-10, Roald

func (h *ApiHandler) AdminGetFavoriteLists(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "admin-get-favorite-lists")
	ctx := gc.Request.Context()
	userUuid := h.userUuid(gc)

	orgUuid := gc.Param("orgUuid")
	if orgUuid == "" {
		apiRequest.Required("orgUuid is required")
		return
	}
	apiRequest.SetMeta("org_uuid", orgUuid)

	query := app.UranusInstance.SqlAdminGetFavoriteLists
	rows, err := h.DbPool.Query(ctx, query, orgUuid, userUuid)
	if err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	var lists []model.FavoriteList

	for rows.Next() {
		var l model.FavoriteList

		if err := rows.Scan(
			&l.Uuid,
			&l.Name,
			&l.Description,
		); err != nil {
			debugf(err.Error())
			apiRequest.InternalServerError()
			return
		}

		lists = append(lists, l)
	}

	if err := rows.Err(); err != nil {
		debugf(err.Error())
		apiRequest.InternalServerError()
		return
	}

	apiRequest.Success(http.StatusOK, lists, "favorite lists loaded successfully")
}
