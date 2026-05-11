package model

import (
	"time"
)

type OrgMember struct {
	UserUuid     string     `json:"user_uuid"`
	Email        string     `json:"email"`
	Username     *string    `json:"username"`
	DisplayName  *string    `json:"display_name"`
	AvatarUrl    *string    `json:"avatar_url"`
	LastActiveAt *time.Time `json:"last_active_at"`
	JoinedAt     time.Time  `json:"joined_at"`
}

type InvitedOrgMember struct {
	UserUuid    string    `json:"user_uuid"`
	InvitedBy   string    `json:"invited_by"`
	InvitedAt   time.Time `json:"invited_at"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name"`
	AvatarUrl   *string   `json:"avatar_url"`
}

type OrgMemberRole struct {
	Uuid        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type OrgMemberLink struct {
	Uuid              string     `json:"uuid"`
	OrgUuid           string     `json:"org_uuid"`
	UserUuid          string     `json:"user_uuid"`
	HasJoined         *bool      `json:"has_joined"`
	InvitedByUserUuid *string    `json:"invited_by_user_uuid"`
	InvitedAt         *time.Time `json:"invited_at"`
	CreatedAt         *time.Time `json:"created_at"`
	ModifiedAt        *time.Time `json:"modified_at"`
}

type OrgListItem struct {
	Uuid                string  `json:"uuid"`
	Name                string  `json:"name"`
	City                *string `json:"city"`
	Country             *string `json:"country"`
	TotalUpcomingEvents int64   `json:"total_upcoming_events"`
	VenueCount          int64   `json:"venue_count"`
	SpaceCount          int64   `json:"space_count"`
	CanEditOrg          bool    `json:"can_edit_org"`
	CanDeleteOrg        bool    `json:"can_delete_org"`
	CanManageTeam       bool    `json:"can_manage_team"`
	LogoUrl             *string `json:"logo_url,omitempty"`
	LightThemeLogoUrl   *string `json:"light_theme_logo_url,omitempty"`
	DarkThemeLogoUrl    *string `json:"dark_theme_logo_url,omitempty"`
}

type OrgPartnerListItem struct {
	Direction         string `json:"direction"`
	OrgUuid           string `json:"org_uuid"`
	OrgName           string `json:"org_name"`
	CanChooseVenue    bool   `json:"can_choose_venue,omitempty"`
	CanChoosePartner  bool   `json:"can_choose_partner,omitempty"`
	CanChoosePromoter bool   `json:"can_choose_promoter,omitempty"`
	CanSeeInsights    bool   `json:"can_see_insights,omitempty"`
	Permissions       int64  `json:"permissions"`
}

type OrgPartnerRequestItem struct {
	OrgUuid   string    `json:"org_uuid"`
	OrgName   string    `json:"org_name"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
	Direction string    `json:"direction"`
	Status    string    `json:"status"`
}
