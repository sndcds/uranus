package model

import "time"

type OrganizationMember struct {
	MemberId     int        `json:"member_id"`
	UserId       int        `json:"user_id"`
	Email        string     `json:"email"`
	UserName     *string    `json:"user_name"`
	DisplayName  *string    `json:"display_name"`
	AvatarUrl    *string    `json:"avatar_url"`
	LastActiveAt *time.Time `json:"last_active_at"`
	JoinedAt     time.Time  `json:"joined_at"`
}

type InvitedOrganizationMember struct {
	UserID    int       `json:"user_id"`
	InvitedBy string    `json:"invited_by"`
	InvitedAt time.Time `json:"invited_at"`
	Email     string    `json:"email"`
	RoleID    int       `json:"role_id"`
	RoleName  string    `json:"role_name"`
}

type OrganizationMemberRole struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
