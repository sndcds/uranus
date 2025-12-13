package app

type Permission uint64

// TODO: Review code

const (
	// Organizer permissions
	PermEditOrganizer          Permission = 1 << 0
	PermDeleteOrganizer        Permission = 1 << 1
	PermChooseAsEventOrganizer Permission = 1 << 2
	PermChooseAsEventPartner   Permission = 1 << 3
	PermReceiveOrganizerMsgs   Permission = 1 << 4
	PermManagePermissions      Permission = 1 << 5
	PermManageTeam             Permission = 1 << 6

	// Venue permissions
	PermAddVenue    Permission = 1 << 8
	PermEditVenue   Permission = 1 << 9
	PermDeleteVenue Permission = 1 << 10
	PermChooseVenue Permission = 1 << 11

	// Space permissions
	PermAddSpace    Permission = 1 << 16
	PermEditSpace   Permission = 1 << 17
	PermDeleteSpace Permission = 1 << 18

	// Event permissions
	PermAddEvent          Permission = 1 << 24
	PermEditEvent         Permission = 1 << 25
	PermDeleteEvent       Permission = 1 << 26
	PermReleaseEvent      Permission = 1 << 27
	PermViewEventInsights Permission = 1 << 28

	PermCombinationAdmin = 0b00011111000001110000111100111111
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
