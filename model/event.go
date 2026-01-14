package model

import (
	"encoding/json"
	"time"
)

type EventType struct {
	TypeID  int `json:"type_id"`
	GenreID int `json:"genre_id"`
}

type EventPlace struct {
	VenueId     int64   `json:"venue_id"`
	VenueName   *string `json:"venue_name"`
	SpaceId     *int64  `json:"space_id"`
	SpaceName   *string `json:"space_name"`
	City        *string `json:"city"`
	CountryCode *string `json:"country_code"`
}

type AdminEvent struct {
	// Core
	EventId     int     `json:"event_id"`
	Title       string  `json:"title"`
	Subtitle    *string `json:"subtitle,omitempty"`
	Description *string `json:"description,omitempty"`
	Summary     *string `json:"summary,omitempty"`

	// Participation / Info
	ParticipationInfo *string `json:"participation_info,omitempty"`
	MeetingPoint      *string `json:"meeting_point,omitempty"`

	MinAge               *int     `json:"min_age,omitempty"`
	MaxAge               *int     `json:"max_age,omitempty"`
	MaxAttendees         *int     `json:"max_attendees,omitempty"`
	PriceTypeId          *int     `json:"price_type_id,omitempty"`
	MinPrice             *float64 `json:"min_price,omitempty"`
	MaxPrice             *float64 `json:"max_price,omitempty"`
	TicketAdvance        *bool    `json:"ticket_advance,omitempty"`
	TicketRequired       *bool    `json:"ticket_required,omitempty"`
	RegistrationRequired *bool    `json:"registration_required,omitempty"`

	CurrencyCode *string `json:"currency_code,omitempty"`
	CurrencyName *string `json:"currency_name,omitempty"`

	OccasionTypeId *int `json:"occasion_type_id,omitempty"`

	// --- URLs ---
	OnlineEventUrl *string `json:"online_event_url,omitempty"`
	SourceUrl      *string `json:"source_url,omitempty"`

	// Media
	Image1Id         *int `json:"image1_id,omitempty"`
	Image2Id         *int `json:"image2_id,omitempty"`
	Image3Id         *int `json:"image3_id,omitempty"`
	Image4Id         *int `json:"image4_id,omitempty"`
	ImageSoMe16To9Id *int `json:"image_some_16_9_id,omitempty"`
	ImageSoMe4To5Id  *int `json:"image_some_4_5_id,omitempty"`
	ImageSoMe9To16Id *int `json:"image_some_9_16_id,omitempty"`
	ImageSoMe1To1Id  *int `json:"image_some_1_1_id,omitempty"`

	// Meta
	Custom *string `json:"custom,omitempty"`
	Style  *string `json:"style,omitempty"`

	ReleaseStatusId int     `json:"release_status_id"`
	ReleaseDate     *string `json:"release_date,omitempty"`

	Languages []string `json:"languages"`
	Tags      []string `json:"tags"`

	// Organization
	OrganizationId   int    `json:"organization_id"`
	OrganizationName string `json:"organization_name"`

	// Venue (main)
	VenueId          *int     `json:"venue_id,omitempty"`
	VenueName        *string  `json:"venue_name,omitempty"`
	VenueStreet      *string  `json:"venue_street,omitempty"`
	VenueHouseNumber *string  `json:"venue_house_number,omitempty"`
	VenuePostalCode  *string  `json:"venue_postal_code,omitempty"`
	VenueCity        *string  `json:"venue_city,omitempty"`
	VenueCountryCode *string  `json:"venue_country_code,omitempty"`
	VenueStateCode   *string  `json:"venue_state_code,omitempty"`
	VenueLon         *float64 `json:"venue_lon,omitempty"`
	VenueLat         *float64 `json:"venue_lat,omitempty"`

	// Space (main)
	SpaceId              *int    `json:"space_id,omitempty"`
	SpaceName            *string `json:"space_name,omitempty"`
	SpaceTotalCapacity   *int    `json:"space_total_capacity,omitempty"`
	SpaceSeatingCapacity *int    `json:"space_seating_capacity,omitempty"`
	SpaceBuildingLevel   *int    `json:"space_building_level,omitempty"`
	SpaceUrl             *string `json:"space_url,omitempty"`

	// Location (custom)
	LocationName        *string `json:"location_name,omitempty"`
	LocationStreet      *string `json:"location_street,omitempty"`
	LocationHouseNumber *string `json:"location_house_number,omitempty"`
	LocationPostalCode  *string `json:"location_postal_code,omitempty"`
	LocationCity        *string `json:"location_city,omitempty"`
	LocationCountryCode *string `json:"location_country_code,omitempty"`
	LocationStateCode   *string `json:"location_state_code,omitempty"`

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
	StartDate *string `json:"start_date,omitempty"`
	StartTime *string `json:"start_time,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
	EndTime   *string `json:"end_time,omitempty"`
	EntryTime *string `json:"entry_time,omitempty"`

	// Meta
	Duration          *int64  `json:"duration,omitempty"`
	AccessibilityInfo *string `json:"accessibility_info,omitempty"`
	VisitorInfoFlags  *int64  `json:"visitor_info_flags,omitempty"`
	DateVenueId       *int    `json:"date_venue_id,omitempty"`

	// Venue (resolved per date)
	VenueId          *int     `json:"venue_id,omitempty"`
	VenueName        *string  `json:"venue_name,omitempty"`
	VenueStreet      *string  `json:"venue_street,omitempty"`
	VenueHouseNumber *string  `json:"venue_house_number,omitempty"`
	VenuePostalCode  *string  `json:"venue_postal_code,omitempty"`
	VenueCity        *string  `json:"venue_city,omitempty"`
	VenueStateCode   *string  `json:"venue_state_code,omitempty"`
	VenueCountryCode *string  `json:"venue_country_code,omitempty"`
	VenueLon         *float64 `json:"venue_lon,omitempty"`
	VenueLat         *float64 `json:"venue_lat,omitempty"`
	VenueUrl         *string  `json:"venue_url,omitempty"`

	// Space (only if venue-bound)
	SpaceId              *int    `json:"space_id,omitempty"`
	SpaceName            *string `json:"space_name,omitempty"`
	SpaceTotalCapacity   *int    `json:"space_total_capacity,omitempty"`
	SpaceSeatingCapacity *int    `json:"space_seating_capacity,omitempty"`
	SpaceBuildingLevel   *int    `json:"space_building_level,omitempty"`
	SpaceUrl             *string `json:"space_url,omitempty"`
}

// EventDatePayload is used for creating and updating event dates.
// StartDate and StartTime are required.
// All other fields are optional and represented as pointers to distinguish
// between "not provided" and zero values.
type EventDatePayload struct {
	StartDate            string  `json:"start_date" binding:"required"`
	StartTime            string  `json:"start_time" binding:"required"`
	EndDate              *string `json:"end_date,omitempty"`
	EndTime              *string `json:"end_time,omitempty"`
	EntryTime            *string `json:"entry_time,omitempty"`
	AllDay               *bool   `json:"all_day,omitempty"`
	Duration             *int    `json:"duration,omitempty"`
	VenueId              *int    `json:"venue_id,omitempty"`
	SpaceId              *int    `json:"space_id,omitempty"`
	VisitorInfoFlags     *int64  `json:"visitor_info_flags,omitempty"`
	TicketLink           *string `json:"ticket_link,omitempty"`
	AvailabilityStatusId *int    `json:"availability_status_id,omitempty"`
	AccessibilityInfo    *string `json:"accessibility_info,omitempty"`
	Custom               *string `json:"custom,omitempty"`
}

type EventDahboardEntry struct {
	EventId               int         `json:"event_id"`
	EventDateId           int         `json:"event_date_id"`
	EventTitle            string      `json:"event_title"`
	EventSubtitle         *string     `json:"event_subtitle"`
	EventOrganizationId   int         `json:"event_organization_id"`
	EventOrganizationName *string     `json:"event_organization_name"`
	StartDate             *string     `json:"start_date"`
	StartTime             *string     `json:"start_time"`
	EndDate               *string     `json:"end_date"`
	EndTime               *string     `json:"end_time"`
	ReleaseStatusId       *int        `json:"release_status_id"`
	ReleaseDate           *string     `json:"release_date"`
	VenueId               *int        `json:"venue_id"`
	VenueName             *string     `json:"venue_name"`
	SpaceId               *int        `json:"space_id,omitempty"`
	SpaceName             *string     `json:"space_name,omitempty"`
	LocationId            *int        `json:"location_id"`
	LocationName          *string     `json:"location_name"`
	ImageId               *int        `json:"image_id"`
	EventTypes            []EventType `json:"event_types"`
	CanEditEvent          bool        `json:"can_edit_event"`
	CanDeleteEvent        bool        `json:"can_delete_event"`
	CanReleaseEvent       bool        `json:"can_release_event"`
	TimeSeriesIndex       int         `json:"time_series_index"`
	TimeSeries            int         `json:"time_series"`
}

// UserEventNotification contains a single event notification
type UserEventNotification struct {
	EventId           int        `json:"event_id"`
	EventTitle        string     `json:"event_title"`
	OrganizationId    int        `json:"organization_id"`
	OrganizationName  *string    `json:"organization_name,omitempty"`
	ReleaseDate       *time.Time `json:"release_date,omitempty"`
	ReleaseStatusId   int        `json:"release_status_id"`
	EarliestEventDate *time.Time `json:"earliest_event_date,omitempty"`
	LatestEventDate   *time.Time `json:"latest_event_date,omitempty"`
	DaysUntilRelease  *int       `json:"days_until_release,omitempty"`
	DaysUntilEvent    *int       `json:"days_until_event,omitempty"`
}
