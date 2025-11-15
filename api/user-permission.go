package api

import "fmt"

type UserPermission int64

const (
	PermissionAdmin            UserPermission = 0b00011111000001110000111100111111
	PermissionManager          UserPermission = 0b00011111000001110000111100111101
	PermissionAssistent        UserPermission = 0b00011111000000100000001000011101
	PermissionBooker           UserPermission = 0b00011111000000000000000000000000
	PermissionEventProofreader UserPermission = 0b00010010000000000000000000000000
	PermissionVenueManager     UserPermission = 0b00011111000000100000001000000000
	PermissionSpaceManager     UserPermission = 0b00000000000000100000000000000000
	PermissionInsightViewer    UserPermission = 0b00010000000000000000000000000000
)

// Organizer permissions (bits 0–3)
const (
	PermissionBitEditOrganizer               uint = 0
	PermissionBitDeleteOrganizer             uint = 1
	PermissionBitChooseAsEventOrganizer      uint = 2
	PermissionBitChooseAsEventPartner        uint = 3
	PermissionBitCanReceiveOrganizerMessages uint = 4
	PermissionBitManagePermissions           uint = 5
)

// Venue permissions (bits 8–11)
const (
	PermissionBitAddVenue    uint = 8
	PermissionBitEditVenue   uint = 9
	PermissionBitDeleteVenue uint = 10
	PermissionBitChooseVenue uint = 11
)

// Space permissions (bits 16–18)
const (
	PermissionBitAddSpace    uint = 16
	PermissionBitEditSpace   uint = 17
	PermissionBitDeleteSpace uint = 18
)

// Event permissions (bits 24–28)
const (
	PermissionBitAddEvent          uint = 24
	PermissionBitEditEvent         uint = 25
	PermissionBitDeleteEvent       uint = 26
	PermissionBitReleaseEvent      uint = 27
	PermissionBitViewEventInsights uint = 28
)

func (p *UserPermission) Set(perm UserPermission) {
	*p = *p | perm
}

func (p *UserPermission) Clear(perm UserPermission) {
	*p = *p &^ perm // &^ is bit clear (AND NOT)
}

func (p UserPermission) Has(perm UserPermission) bool {
	return p&perm != 0
}

func (p *UserPermission) SetBit(bit uint) {
	if bit > 63 {
		return // or panic if you want
	}
	*p |= 1 << bit
}

func (p *UserPermission) ClearBit(bit uint) {
	if bit > 63 {
		return
	}
	*p &^= 1 << bit // AND NOT clears the bit
}

func (p UserPermission) HasBit(bit uint) bool {
	if bit > 63 {
		return false
	}
	return p&(1<<bit) != 0
}

func (p UserPermission) Binary() string {
	return fmt.Sprintf("%064b", int64(p))
}
