package model

type Venue struct {
	Uuid         string           `json:"uuid"`
	OrgUuid      string           `json:"org_uuid"`
	Name         string           `json:"name"`
	Description  *string          `json:"description,omitempty"`
	Type         *string          `json:"type,omitempty"`
	OpenedAt     *string          `json:"opened_at,omitempty"`
	ClosedAt     *string          `json:"closed_at,omitempty"`
	ContactEmail *string          `json:"contact_email,omitempty"`
	ContactPhone *string          `json:"contact_phone,omitempty"`
	WebLink      *string          `json:"web_link,omitempty"`
	Street       *string          `json:"street,omitempty"`
	HouseNumber  *string          `json:"house_number,omitempty"`
	PostalCode   *string          `json:"postal_code,omitempty"`
	City         *string          `json:"city,omitempty"`
	State        *string          `json:"state,omitempty"`
	Country      *string          `json:"country,omitempty"`
	Lon          *float64         `json:"lon,omitempty"`
	Lat          *float64         `json:"lat,omitempty"`
	Images       map[string]Image `json:"images,omitempty"`
}
