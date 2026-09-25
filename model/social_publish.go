package model

type SocialPublishResult struct {
	RemotePostID string `json:"remote_post_id"`
}

type SocialPostPublish struct {
	PostUuid string                `json:"post_uuid"`
	Results  []SocialTargetPublish `json:"results"`
}

type SocialTargetPublish struct {
	TargetUuid        string  `json:"target_uuid"`
	SocialAccountUuid string  `json:"social_account_uuid"`
	Platform          string  `json:"platform"`
	Status            string  `json:"status"`
	RemotePostID      *string `json:"remote_post_id,omitempty"`
	Error             *string `json:"error,omitempty"`
}
