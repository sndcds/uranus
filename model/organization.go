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

type OrganizationMemberLink struct {
	Id              int        `json:"id"`
	OrganizationId  int        `json:"organization_id"`
	UserId          int        `json:"user_id"`
	HasJoined       *bool      `json:"has_joined"`
	InvitedByUserId *int       `json:"invited_by_user_id"`
	InvitedAt       *time.Time `json:"invited_at"`
	CreatedAt       *time.Time `json:"created_at"`
	ModifiedAt      *time.Time `json:"modified_at"`
}

type OrganizationDashboardEntry struct {
	OrganizationId          int64   `json:"organization_id"`
	OrganizationName        string  `json:"organization_name"`
	OrganizationCity        *string `json:"organization_city"`
	OrganizationCountryCode *string `json:"organization_country_code"`
	TotalUpcomingEvents     int64   `json:"total_upcoming_events"`
	VenueCount              int64   `json:"venue_count"`
	SpaceCount              int64   `json:"space_count"`
	CanEditOrganization     bool    `json:"can_edit_organization"`
	CanDeleteOrganization   bool    `json:"can_delete_organization"`
	CanManageTeam           bool    `json:"can_manage_team"`
}
