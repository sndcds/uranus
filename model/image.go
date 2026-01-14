package model

type ImageMeta struct {
	Id           *int           `json:"id"`
	FileName     *string        `json:"file_name"`
	Width        *int           `json:"width"`
	Height       *int           `json:"height"`
	MimeType     *string        `json:"mime_type"`
	AltText      *string        `json:"alt_text"`
	Description  *string        `json:"description"`
	LicenseID    *int           `json:"license_id"`
	Exif         map[string]any `json:"exif"`
	Expiration   *string        `json:"expiration_date"`
	CreatorName  *string        `json:"creator_name"`
	Copyright    *string        `json:"copyright"`
	FocusX       *float64       `json:"focus_x"`
	FocusY       *float64       `json:"focus_y"`
	MarginLeft   *int           `json:"margin_left"`
	MarginRight  *int           `json:"margin_right"`
	MarginTop    *int           `json:"margin_top"`
	MarginBottom *int           `json:"margin_bottom"`
}
