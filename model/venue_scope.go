package model

// VenueScope is the authoritative classification stored in venue.scope.
// It is independent of organization IDs and permissions.
type VenueScope string

const (
	// VenueScopeOrganization is local to its organization:
	// "Provisorischer Ort (nicht eigener Ort)" in the admin.
	VenueScopeOrganization VenueScope = "organization"
	// VenueScopeShared is a standalone / reusable venue:
	// "Eigener Ort" in the admin.
	VenueScopeShared VenueScope = "shared"
)

// IsValid deliberately does not normalize case or whitespace. Request handlers
// retain their existing whitespace trimming before validating.
func (s VenueScope) IsValid() bool {
	return s == VenueScopeOrganization || s == VenueScopeShared
}
