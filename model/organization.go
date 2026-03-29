package model

import "time"

type OrganizationMember struct {
	UserUuid     string     `json:"user_uuid"`
	Email        string     `json:"email"`
	Username     *string    `json:"username"`
	DisplayName  *string    `json:"display_name"`
	AvatarUrl    *string    `json:"avatar_url"`
	LastActiveAt *time.Time `json:"last_active_at"`
	JoinedAt     time.Time  `json:"joined_at"`
}

type InvitedOrganizationMember struct {
	UserUuid  string    `json:"user_uuid"`
	InvitedBy string    `json:"invited_by"`
	InvitedAt time.Time `json:"invited_at"`
	Email     string    `json:"email"`
}

type OrganizationMemberRole struct {
	Uuid        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type OrganizationMemberLink struct {
	Uuid              string     `json:"uuid"`
	OrgUuid           string     `json:"org_uuid"`
	UserUuid          string     `json:"user_uuid"`
	HasJoined         *bool      `json:"has_joined"`
	InvitedByUserUuid *string    `json:"invited_by_user_uuid"`
	InvitedAt         *time.Time `json:"invited_at"`
	CreatedAt         *time.Time `json:"created_at"`
	ModifiedAt        *time.Time `json:"modified_at"`
}

type OrganizationDashboardEntry struct {
	OrgUuid             string  `json:"org_uuid"`
	OrgName             string  `json:"org_name"`
	OrgCity             *string `json:"org_city"`
	OrgCountry          *string `json:"org_country"`
	TotalUpcomingEvents int64   `json:"total_upcoming_events"`
	VenueCount          int64   `json:"venue_count"`
	SpaceCount          int64   `json:"space_count"`
	CanEditOrg          bool    `json:"can_edit_org"`
	CanDeleteOrg        bool    `json:"can_delete_org"`
	CanManageTeam       bool    `json:"can_manage_team"`
	MainLogoUuid        *string `json:"main_logo_uuid"`
	DarkThemeLogoUuid   *string `json:"dark_theme_logo_uuid"`
	LightThemeLogoUuid  *string `json:"light_theme_logo_uuid"`
}
