package model

type WebLink struct {
	Id    int     `json:"id"`
	Title *string `json:"title,omitempty"`
	Type  *string `json:"type,omitempty"`
	Url   string  `json:"url"`
}
