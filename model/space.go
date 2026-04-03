package model

type Space struct {
	Uuid                 string   `json:"uuid"`
	Name                 string   `json:"name"`
	Description          *string  `json:"description,omitempty"`
	SpaceType            *string  `json:"space_type,omitempty"`
	BuildingLevel        *int     `json:"building_level,omitempty"`
	AreaSqm              *float64 `json:"area_sqm,omitempty"`
	TotalCapacity        *int     `json:"total_capacity,omitempty"`
	SeatingCapacity      *int     `json:"seating_capacity,omitempty"`
	WebLink              *string  `json:"web_link,omitempty"`
	AccessibilityFlags   *string  `json:"accessibility_flags"`
	AccessibilitySummary *string  `json:"accessibility_summary"`
}
