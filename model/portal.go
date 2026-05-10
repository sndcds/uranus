package model

import "encoding/json"

type AdminListPortal struct {
	Uuid            string  `json:"uuid"`
	Name            *string `json:"name"`
	Description     *string `json:"description,omitempty"`
	CanEditPortal   bool    `json:"can_edit_portal"`
	CanDeletePortal bool    `json:"can_delete_portal"`
}

type Portal struct {
	Uuid              string          `json:"uuid"`
	OrgUuid           string          `json:"org_uuid"`
	Name              *string         `json:"name"`
	Description       *string         `json:"description,omitempty"`
	SpatialFilterMode *string         `json:"spatial_filter_mode,omitempty"`
	PreFilter         json.RawMessage `json:"pre_filter,omitempty"`
	Geometry          json.RawMessage `json:"geometry,omitempty"`
	Style             json.RawMessage `json:"style,omitempty"`
}
