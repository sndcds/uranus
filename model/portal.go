package model

type AdminListPortal struct {
	Uuid            string  `json:"uuid"`
	Name            *string `json:"name"`
	Description     *string `json:"description,omitempty"`
	CanEditPortal   bool    `json:"can_edit_portal"`
	CanDeletePortal bool    `json:"can_delete_portal"`
}
