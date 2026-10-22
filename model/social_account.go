package model

import "time"

// SocialAccount is response metadata only. Credential values are never selected
// into this type, including when returning a newly created or updated account.
type SocialAccount struct {
	Uuid              string     `json:"uuid"`
	OrgUuid           string     `json:"org_uuid"`
	Platform          string     `json:"platform"`
	Name              string     `json:"name"`
	RemoteAccountID   string     `json:"remote_account_id"`
	RemoteAccountName *string    `json:"remote_account_name"`
	TokenExpiresAt    *time.Time `json:"token_expires_at"`
	BaseURL           *string    `json:"base_url"`
	Enabled           bool       `json:"enabled"`
	HasAccessToken    bool       `json:"has_access_token"`
	HasRefreshToken   bool       `json:"has_refresh_token"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
