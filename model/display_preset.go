package model

type DisplayPreset struct {
	Uuid        string  `json:"uuid"`
	OrgUuid     string  `json:"org_uuid"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	DisplayMode *string `json:"display_mode,omitempty"`
	Options     *string `json:"options,omitempty"`
	Code        *string `json:"code,omitempty"`
}
