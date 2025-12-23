package app

type Permission uint64

// TODO: Review code

const (
	// Organization permissions
	PermBitEditOrganization          = 0
	PermBitDeleteOrganization        = 1
	PermBitChooseAsEventOrganization = 2
	PermBitChooseAsEventPartner      = 3
	PermBitReceiveOrganizationMsgs   = 4
	PermBitManagePermissions         = 5
	PermBitManageTeam                = 6

	// Venue permissions
	PermBitAddVenue    = 8
	PermBitEditVenue   = 9
	PermBitDeleteVenue = 10
	PermBitChooseVenue = 11

	// Space permissions
	PermBitAddSpace    = 16
	PermBitEditSpace   = 17
	PermBitDeleteSpace = 18

	// Event permissions
	PermBitAddEvent          = 24
	PermBitEditEvent         = 25
	PermBitDeleteEvent       = 26
	PermBitReleaseEvent      = 27
	PermBitViewEventInsights = 28
)

const (
	// Organization permissions
	PermEditOrganization          Permission = 1 << PermBitEditOrganization
	PermDeleteOrganization        Permission = 1 << PermBitDeleteOrganization
	PermChooseAsEventOrganization Permission = 1 << PermBitChooseAsEventOrganization
	PermChooseAsEventPartner      Permission = 1 << PermBitChooseAsEventPartner
	PermReceiveOrganizationMsgs   Permission = 1 << PermBitReceiveOrganizationMsgs
	PermManagePermissions         Permission = 1 << PermBitManagePermissions
	PermManageTeam                Permission = 1 << PermBitManageTeam

	// Venue permissions
	PermAddVenue    Permission = 1 << PermBitAddVenue
	PermEditVenue   Permission = 1 << PermBitEditVenue
	PermDeleteVenue Permission = 1 << PermBitDeleteVenue
	PermChooseVenue Permission = 1 << PermBitChooseVenue

	// Space permissions
	PermAddSpace    Permission = 1 << PermBitAddSpace
	PermEditSpace   Permission = 1 << PermBitEditSpace
	PermDeleteSpace Permission = 1 << PermBitDeleteSpace

	// Event permissions
	PermAddEvent          Permission = 1 << PermBitAddEvent
	PermEditEvent         Permission = 1 << PermBitEditEvent
	PermDeleteEvent       Permission = 1 << PermBitDeleteEvent
	PermReleaseEvent      Permission = 1 << PermBitReleaseEvent
	PermViewEventInsights Permission = 1 << PermBitViewEventInsights

	PermCombinationAdmin = 0b00011111000001110000111101111111
)

func (p Permission) Has(flag Permission) bool {
	return p&flag != 0
}

func (p *Permission) Add(flag Permission) {
	*p |= flag
}

func (p *Permission) Remove(flag Permission) {
	*p &^= flag
}

// HasAll checks if all bits in 'mask' are set in 'p'
func (p Permission) HasAll(mask Permission) bool {
	return p&mask == mask
}

// HasAny checks if at least one bit in 'mask' is set in 'p'
func (p Permission) HasAny(mask Permission) bool {
	return p&mask != 0
}
