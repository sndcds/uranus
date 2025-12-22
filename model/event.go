package model

import "encoding/json"

type AdminEvent struct {
	// Core
	EventId     int     `json:"event_id"`
	Title       string  `json:"title"`
	Subtitle    *string `json:"subtitle"`
	Description *string `json:"description"`
	TeaserText  *string `json:"teaser_text"`

	// Participation / Info
	ParticipationInfo *string `json:"participation_info"`
	MeetingPoint      *string `json:"meeting_point"`

	MinAge               *int     `json:"min_age"`
	MaxAge               *int     `json:"max_age"`
	MaxAttendees         *int     `json:"max_attendees"`
	PriceTypeId          *int     `json:"price_type_id"`
	MinPrice             *float64 `json:"min_price"`
	MaxPrice             *float64 `json:"max_price"`
	TicketAdvance        *bool    `json:"ticket_advance"`
	TicketRequired       *bool    `json:"ticket_required"`
	RegistrationRequired *bool    `json:"registration_required"`

	CurrencyCode *string `json:"currency_code"`
	CurrencyName string  `json:"currency_name"`

	OccasionTypeId *int `json:"occasion_type_id"`

	// --- URLs ---
	OnlineEventUrl *string `json:"online_event_url"`
	SourceUrl      *string `json:"source_url"`

	// Media
	Image1Id         *int `json:"image1_id"`
	Image2Id         *int `json:"image2_id"`
	Image3Id         *int `json:"image3_id"`
	Image4Id         *int `json:"image4_id"`
	ImageSoMe16To9Id *int `json:"image_some_16_9_id"`
	ImageSoMe4To5Id  *int `json:"image_some_4_5_id"`
	ImageSoMe9To16Id *int `json:"image_some_9_16_id"`
	ImageSoMe1To1Id  *int `json:"image_some_1_1_id"`

	// Meta
	Custom *string `json:"custom"`
	Style  *string `json:"style"`

	ReleaseStatusId int     `json:"release_status_id"`
	ReleaseDate     *string `json:"release_date"`

	Languages []string `json:"languages"`
	Tags      []string `json:"tags"`

	// Organization
	OrganizationId   int    `json:"organization_id"`
	OrganizationName string `json:"organization_name"`

	// Venue (main)
	VenueId          *int     `json:"venue_id"`
	VenueName        *string  `json:"venue_name"`
	VenueStreet      *string  `json:"venue_street"`
	VenueHouseNumber *string  `json:"venue_house_number"`
	VenuePostalCode  *string  `json:"venue_postal_code"`
	VenueCity        *string  `json:"venue_city"`
	VenueCountryCode *string  `json:"venue_country_code"`
	VenueStateCode   *string  `json:"venue_state_code"`
	VenueLon         *float64 `json:"venue_lon"`
	VenueLat         *float64 `json:"venue_lat"`

	// Space (main)
	SpaceId              *int    `json:"space_id"`
	SpaceName            *string `json:"space_name"`
	SpaceTotalCapacity   *int    `json:"space_total_capacity"`
	SpaceSeatingCapacity *int    `json:"space_seating_capacity"`
	SpaceBuildingLevel   *int    `json:"space_building_level"`
	SpaceUrl             *string `json:"space_url"`

	// Location (custom)
	LocationName        *string `json:"location_name"`
	LocationStreet      *string `json:"location_street"`
	LocationHouseNumber *string `json:"location_house_number"`
	LocationPostalCode  *string `json:"location_postal_code"`
	LocationCity        *string `json:"location_city"`
	LocationCountryCode *string `json:"location_country_code"`
	LocationStateCode   *string `json:"location_state_code"`

	// Aggregates
	EventTypes json.RawMessage `json:"event_types"`
	EventUrls  json.RawMessage `json:"event_urls"`

	// Populated separately
	EventDates []AdminEventDate `json:"event_dates"`
}

type AdminEventDate struct {
	// Core
	EventDateId int `json:"event_date_id"`
	EventId     int `json:"event_id"`

	// Dates & Times (already formatted in SQL)
	StartDate *string `json:"start_date"`
	StartTime *string `json:"start_time"`
	EndDate   *string `json:"end_date"`
	EndTime   *string `json:"end_time"`
	EntryTime *string `json:"entry_time"`

	// Meta
	Duration          *int64  `json:"duration"`
	AccessibilityInfo *string `json:"accessibility_info"`
	VisitorInfoFlags  *int64  `json:"visitor_info_flags"`
	DateVenueId       *int    `json:"date_venue_id"`

	// Venue (resolved per date)
	VenueId          *int     `json:"venue_id"`
	VenueName        *string  `json:"venue_name"`
	VenueStreet      *string  `json:"venue_street"`
	VenueHouseNumber *string  `json:"venue_house_number"`
	VenuePostalCode  *string  `json:"venue_postal_code"`
	VenueCity        *string  `json:"venue_city"`
	VenueStateCode   *string  `json:"venue_state_code"`
	VenueCountryCode *string  `json:"venue_country_code"`
	VenueLon         *float64 `json:"venue_lon"`
	VenueLat         *float64 `json:"venue_lat"`
	VenueUrl         *string  `json:"venue_url"`

	// Space (only if venue-bound)
	SpaceId              *int    `json:"space_id"`
	SpaceName            *string `json:"space_name"`
	SpaceTotalCapacity   *int    `json:"space_total_capacity"`
	SpaceSeatingCapacity *int    `json:"space_seating_capacity"`
	SpaceBuildingLevel   *int    `json:"space_building_level"`
	SpaceUrl             *string `json:"space_url"`
}
