package model

import "time"

type SocialPublication struct {
	Uuid                 string     `json:"uuid"`
	SocialPostUuid       string     `json:"social_post_uuid"`
	SocialPostTargetUuid string     `json:"social_post_target_uuid"`
	SocialAccountUuid    string     `json:"social_account_uuid"`
	Platform             string     `json:"platform"`
	PublicationSource    string     `json:"publication_source"`
	ContentFingerprint   *string    `json:"content_fingerprint"`
	RenderedText         *string    `json:"rendered_text"`
	RenderedImageURL     *string    `json:"rendered_image_url"`
	RenderedImageAlt     *string    `json:"rendered_image_alt"`
	Status               string     `json:"status"`
	RemotePostID         *string    `json:"remote_post_id"`
	Error                *string    `json:"error"`
	StartedAt            time.Time  `json:"started_at"`
	FinishedAt           *time.Time `json:"finished_at"`
	ReconciledAt         *time.Time `json:"reconciled_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
