package model

type ImageLicense struct {
	Id        int     `json:"id"`
	ShortName *string `json:"short_name,omitempty"`
	Name      *string `json:"name,omitempty"`
	Url       *string `json:"url,omitempty"`
}

type Image struct {
	Id          int           `json:"id"`
	Identifier  string        `json:"identifier"`
	Url         string        `json:"url"`
	Alt         *string       `json:"alt,omitempty"`
	Creator     *string       `json:"creator,omitempty"`
	Copyright   *string       `json:"copyright,omitempty"`
	Description *string       `json:"description,omitempty"`
	License     *ImageLicense `json:"license,omitempty"`
	FocusX      *float64      `json:"focus_x,omitempty"`
	FocusY      *float64      `json:"focus_y,omitempty"`
}
