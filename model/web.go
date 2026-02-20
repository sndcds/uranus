package model

type WebLink struct {
	Label *string `json:"label,omitempty"`
	Type  *string `json:"type,omitempty"`
	Url   string  `json:"url"`
}
