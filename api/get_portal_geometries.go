package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/grains/grains_api"
	"github.com/sndcds/uranus/app"
)

type portalGeometry struct {
	Uuid        string            `json:"uuid"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Order       int               `json:"order"`
	AdminLevel  *int              `json:"admin_level"`
	Type        *string           `json:"type"`
	Geometry    json.RawMessage   `json:"geometry"`
	Childs      []*portalGeometry `json:"childs"`
	ParentUuid  *string           `json:"-"`
}

func (h *ApiHandler) GetPortalGeometries(gc *gin.Context) {
	apiRequest := grains_api.NewRequest(gc, "get-portal-geometries")
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

	rows, err := h.DbPool.Query(gc.Request.Context(), app.UranusInstance.SqlGetPortalGeometries, portalUuid)
	if err != nil {
		debugf("get portal geometries: %v", err)
		apiRequest.InternalServerError()
		return
	}
	defer rows.Close()

	portalExists := false
	geometries := make([]*portalGeometry, 0)
	for rows.Next() {
		portalExists = true
		var geometryUuid *string
		geometry := &portalGeometry{}
		if err := rows.Scan(
			&geometryUuid,
			&geometry.ParentUuid,
			&geometry.Name,
			&geometry.Description,
			&geometry.Order,
			&geometry.AdminLevel,
			&geometry.Type,
			&geometry.Geometry,
		); err != nil {
			debugf("scan portal geometry: %v", err)
			apiRequest.InternalServerError()
			return
		}
		if geometryUuid != nil {
			geometry.Uuid = *geometryUuid
			geometries = append(geometries, geometry)
		}
	}
	if err := rows.Err(); err != nil {
		debugf("read portal geometries: %v", err)
		apiRequest.InternalServerError()
		return
	}
	if !portalExists {
		apiRequest.NotFound("portal not found")
		return
	}

	geometry, err := buildPortalGeometryTree(geometries)
	if err != nil {
		debugf("build portal geometry tree for %s: %v", portalUuid, err)
		apiRequest.InternalServerError()
		return
	}
	apiRequest.Success(http.StatusOK, gin.H{"geometry": geometry})
}

// The query orders all rows by order, name and UUID. Appending in that order
// preserves the same deterministic ordering for siblings at every depth.
func buildPortalGeometryTree(geometries []*portalGeometry) (*portalGeometry, error) {
	byUuid := make(map[string]*portalGeometry, len(geometries))
	var root *portalGeometry
	for _, geometry := range geometries {
		geometry.Childs = make([]*portalGeometry, 0)
		byUuid[geometry.Uuid] = geometry
		if geometry.ParentUuid == nil {
			if root != nil {
				return nil, fmt.Errorf("multiple main geometries")
			}
			root = geometry
		}
	}
	if len(geometries) == 0 {
		return nil, nil
	}
	if root == nil {
		return nil, fmt.Errorf("main geometry missing")
	}
	for _, geometry := range geometries {
		if geometry.ParentUuid == nil {
			continue
		}
		parent := byUuid[*geometry.ParentUuid]
		if parent == nil {
			return nil, fmt.Errorf("parent of geometry %s is outside the portal", geometry.Uuid)
		}
		parent.Childs = append(parent.Childs, geometry)
	}

	// With one parent per node, any cycle is disconnected from the root.
	// Check reachability before JSON encoding to reject such invalid hierarchies.
	visited := 0
	pending := []*portalGeometry{root}
	for len(pending) > 0 {
		geometry := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		visited++
		pending = append(pending, geometry.Childs...)
	}
	if visited != len(geometries) {
		return nil, fmt.Errorf("geometry hierarchy contains a cycle")
	}
	return root, nil
}
