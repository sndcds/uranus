package model

// RenderedPost is content ready for preview, independent of publication state.
type RenderedPost struct {
	Platform string  `json:"platform"`
	Text     string  `json:"text"`
	ImageURL string  `json:"image_url,omitempty"`
	ImageAlt *string `json:"image_alt,omitempty"`
	URL      string  `json:"url,omitempty"`
}

type SocialPostPreview struct {
	PostUuid string                `json:"post_uuid"`
	Previews []SocialTargetPreview `json:"previews"`
}

type SocialTargetPreview struct {
	TargetUuid string `json:"target_uuid"`
	RenderedPost
}
