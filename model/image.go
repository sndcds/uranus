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

type UpsertImageResultData struct {
	HttpStatus        int    `json:"http_status"`
	Message           string `json:"message"`
	FileReplaced      bool   `json:"file_replaced"`
	CacheFilesRemoved int    `json:"cache_files_removed"`
	ImageId           int    `json:"image_id"`
	ImageIdentifier   string `json:"image_identifier"`
}
