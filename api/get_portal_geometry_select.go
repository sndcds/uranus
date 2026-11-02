package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

type portalGeometrySelectItem struct {
	Name         string `json:"name"`
	Order        int    `json:"order"`
	GroupID      *int   `json:"group_id"`
	AdminLevel   *int   `json:"admin_level"`
	GeometryUuid string `json:"geometry_uuid"`
}

func (h *ApiHandler) GetPortalGeometrySelect(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-portal-geometry-select")
	portalUuid := gc.Param("uuid")
	if portalUuid == "" {
		apiRequest.Required("portal uuid is required")
		return
	}
	if !app.ValidUuid(portalUuid) {
		apiRequest.Error(http.StatusBadRequest, "invalid portal uuid")
		return
	}
	apiRequest.SetMeta("portal_uuid", portalUuid)

	var groupID *int
	query := gc.Request.URL.Query()
	if query.Has("group") {
		value, err := strconv.ParseInt(query.Get("group"), 10, 32)
		if err != nil {
			apiRequest.Error(http.StatusBadRequest, "group must be a valid integer")
			return
		}
		group := int(value)
		groupID = &group
	}

	rows, err := h.DbPool.Query(gc.Request.Context(), app.UranusInstance.SqlGetPortalGeometrySelect, portalUuid, groupID)
	if err != nil {
		debugf("get portal geometry select: %v", err)
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	portalExists := false
	items := make([]portalGeometrySelectItem, 0)
	for rows.Next() {
		portalExists = true
		var name, geometryUuid *string
		var item portalGeometrySelectItem
		if err := rows.Scan(&name, &item.Order, &item.GroupID, &item.AdminLevel, &geometryUuid); err != nil {
			debugf("scan portal geometry select: %v", err)
			apiRequest.InternalServerError()
			return
		}
		if geometryUuid != nil {
			item.Name = *name
			item.GeometryUuid = *geometryUuid
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		debugf("read portal geometry select: %v", err)
		apiRequest.InternalServerError()
		return
	}
	if !portalExists {
		apiRequest.NotFound("portal not found")
		return
	}

	apiRequest.Success(http.StatusOK, gin.H{"select": items})
}
