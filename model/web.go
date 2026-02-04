package model

type WebLink struct {
	Label *string `json:"label,omitempty"`
	Type  *int    `json:"type,omitempty"`
	Url   string  `json:"url"`
}
