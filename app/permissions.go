package app

type Permissions uint64

// TODO: Review code

const (
	// Organization permissions
	UserPermBitEditOrg              = 0
	UserPermBitDeleteOrg            = 1
	UserPermBitChooseAsEventOrg     = 2
	UserPermBitChooseAsEventPartner = 3
	UserPermBitReceiveOrgMsgs       = 4
	UserPermBitManagePermissions    = 5
	UserPermBitManageTeam           = 6

	// Venue permissions
	UserPermBitAddVenue    = 8
	UserPermBitEditVenue   = 9
	UserPermBitDeleteVenue = 10
	UserPermBitChooseVenue = 11

	// Space permissions
	UserPermBitAddSpace    = 16
	UserPermBitEditSpace   = 17
	UserPermBitDeleteSpace = 18

	// Event permissions
	UserPermBitAddEvent          = 24
	UserPermBitEditEvent         = 25
	UserPermBitDeleteEvent       = 26
	UserPermBitReleaseEvent      = 27
	UserPermBitViewEventInsights = 28

	OrgPermBitChooseVenue    = 0
	OrgPermBitChoosePartner  = 1
	OrgPermBitChoosePromoter = 2
	OrgPermBitSeeInsights    = 8
)

const (
	// Organization permissions
	UserPermEditOrg              Permissions = 1 << UserPermBitEditOrg
	UserPermDeleteOrg            Permissions = 1 << UserPermBitDeleteOrg
	UserPermChooseAsEventOrg     Permissions = 1 << UserPermBitChooseAsEventOrg
	UserPermChooseAsEventPartner Permissions = 1 << UserPermBitChooseAsEventPartner
	UserPermReceiveOrgMsgs       Permissions = 1 << UserPermBitReceiveOrgMsgs
	UserPermManagePermissions    Permissions = 1 << UserPermBitManagePermissions
	UserPermManageTeam           Permissions = 1 << UserPermBitManageTeam

	// Venue permissions
	UserPermAddVenue    Permissions = 1 << UserPermBitAddVenue
	UserPermEditVenue   Permissions = 1 << UserPermBitEditVenue
	UserPermDeleteVenue Permissions = 1 << UserPermBitDeleteVenue
	UserPermChooseVenue Permissions = 1 << UserPermBitChooseVenue

	// Space permissions
	UserPermAddSpace    Permissions = 1 << UserPermBitAddSpace
	UserPermEditSpace   Permissions = 1 << UserPermBitEditSpace
	UserPermDeleteSpace Permissions = 1 << UserPermBitDeleteSpace

	// Event permissions
	UserPermAddEvent          Permissions = 1 << UserPermBitAddEvent
	UserPermEditEvent         Permissions = 1 << UserPermBitEditEvent
	UserPermDeleteEvent       Permissions = 1 << UserPermBitDeleteEvent
	UserPermReleaseEvent      Permissions = 1 << UserPermBitReleaseEvent
	UserPermViewEventInsights Permissions = 1 << UserPermBitViewEventInsights

	UserPermCombinationAdmin = 0b00011111000001110000111101111111

	OrgPermChooseVenue    Permissions = 1 << OrgPermBitChooseVenue
	OrgPermChoosePartner  Permissions = 1 << OrgPermBitChoosePartner
	OrgPermChoosePromoter Permissions = 1 << OrgPermBitChoosePromoter
	OrgPermSeeInsights    Permissions = 1 << OrgPermBitSeeInsights
)

func (p Permissions) Has(flag Permissions) bool {
	return p&flag != 0
}

func (p *Permissions) Add(flag Permissions) {
	*p |= flag
}

func (p *Permissions) Remove(flag Permissions) {
	*p &^= flag
}

// HasAll checks if all bits in 'mask' are set in 'p'
func (p Permissions) HasAll(mask Permissions) bool {
	return p&mask == mask
}

// HasAny checks if at least one bit in 'mask' is set in 'p'
func (p Permissions) HasAny(mask Permissions) bool {
	return p&mask != 0
}
