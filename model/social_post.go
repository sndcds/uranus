package model

import "time"

type SocialPost struct {
	Uuid       string             `json:"uuid"`
	OrgUuid    string             `json:"org_uuid"`
	SourceType string             `json:"source_type"`
	SourceUuid string             `json:"source_uuid"`
	CreatedBy  *string            `json:"created_by"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	Targets    []SocialPostTarget `json:"targets"`
}

type SocialPostTarget struct {
	Uuid              string     `json:"uuid"`
	SocialAccountUuid string     `json:"social_account_uuid"`
	Status            string     `json:"status"`
	ScheduledAt       *time.Time `json:"scheduled_at"`
	PublishedAt       *time.Time `json:"published_at"`
	RemotePostID      *string    `json:"remote_post_id"`
	Error             *string    `json:"error"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
