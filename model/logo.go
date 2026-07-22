package model

type Logo struct {
	Uuid   string  `json:"uuid"`
	Url    string  `json:"url"`
	Alt    *string `json:"alt,omitempty"`
	Width  *int32  `json:"width,omitempty"`
	Height *int32  `json:"height,omitempty"`
}
