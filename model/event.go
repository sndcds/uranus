package model

import (
	"time"
)

type TicketFlag string

const (
	AdvanceTicket        TicketFlag = "advance_ticket"
	TicketRequired       TicketFlag = "ticket_required"
	OnSiteTicketSales    TicketFlag = "on_site_ticket_sales"
	RegistrationRequired TicketFlag = "registration_required"
)

type PriceType string

const (
	NotSpecified PriceType = "not_specified"
	RegularPrice PriceType = "regular_price"
	Free         PriceType = "free"
	Donation     PriceType = "donation"
	TieredPrices PriceType = "tiered_prices"
)

type EventType struct {
	Type      int     `json:"type_id"`
	TypeName  *string `json:"type_name,omitempty"`
	Genre     int     `json:"genre_id"`
	GenreName *string `json:"genre_name,omitempty"`
}

type EventVenueInfo struct {
	VenueId   int64   `json:"venue_id"`
	VenueName *string `json:"venue_name"`
	SpaceId   *int64  `json:"space_id"`
	SpaceName *string `json:"space_name"`
	City      *string `json:"city"`
	Country   *string `json:"country"`
}

type EventDate struct {
	Id                   int      `json:"id"`
	EventId              int      `json:"event_id"`
	EventReleaseStatus   string   `json:"release_status"`
	StartDate            string   `json:"start_date"`
	StartTime            string   `json:"start_time"`
	EndDate              *string  `json:"end_date,omitempty"`
	EndTime              *string  `json:"end_time,omitempty"`
	EntryTime            *string  `json:"entry_time,omitempty"`
	Duration             *int     `json:"duration,omitempty"`
	AccessibilityFlags   *string  `json:"accessibility_flags,omitempty"`
	AccessibilitySummary *string  `json:"accessibility_summary,omitempty"`
	AccessibilityInfo    *string  `json:"accessibility_info,omitempty"`
	LocationName         *string  `json:"location,omitempty"`
	Street               *string  `json:"street,omitempty"`
	HouseNumber          *string  `json:"house_number,omitempty"`
	PostalCode           *string  `json:"postal_code,omitempty"`
	City                 *string  `json:"city,omitempty"`
	Country              *string  `json:"country,omitempty"`
	State                *string  `json:"state,omitempty"`
	Lon                  *float32 `json:"lon,omitempty"`
	Lat                  *float32 `json:"lat,omitempty"`
	TotalCapacity        *int     `json:"total_capacity,omitempty"`
	SeatingCapacity      *int     `json:"seating_capacity,omitempty"`
	BuildingLevel        *int     `json:"building_level,omitempty"`
	VenueId              *int     `json:"venue_id"`
	VenueWebsiteUrl      *string  `json:"venue_website,omitempty"`
	VenueLogoImageId     *int     `json:"venue_logo_id,omitempty"`
	VenueLogoUrl         *string  `json:"venue_logo_url,omitempty"`
	SpaceId              *int     `json:"space_id,omitempty"`
	SpaceName            *string  `json:"space_name,omitempty"`
	SpaceWebsiteUrl      *string  `json:"space_website,omitempty"`
}

type EventDetails struct {
	Id               int         `json:"id"`
	Title            string      `json:"title"`
	Subtitle         *string     `json:"subtitle,omitempty"`
	Description      *string     `json:"description,omitempty"`
	Summary          *string     `json:"summary,omitempty"`
	Languages        []string    `json:"languages,omitempty"`
	Tags             []string    `json:"tags,omitempty"`
	OrganizationId   int         `json:"organization_id"`
	OrganizationName string      `json:"organization_name"`
	OrganizationUrl  *string     `json:"organization_website,omitempty"`
	Image            *Image      `json:"image,omitempty"`       // nested struct for the JSON image
	EventTypes       []EventType `json:"event_types,omitempty"` // typed slice
	EventLinks       []WebLink   `json:"event_links,omitempty"` // typed slice
	Date             *EventDate  `json:"date,omitempty"`
	FurtherDates     []EventDate `json:"further_dates,omitempty"`
}

type AdminEvent struct {
	Id                   int              `json:"id"`
	ExternalId           *string          `json:"external_id,omitempty"`
	SourceUrl            *string          `json:"source_url,omitempty"`
	ReleaseStatus        string           `json:"release_status"`
	ReleaseDate          *string          `json:"release_date,omitempty"`
	ContentLanguage      *string          `json:"content_language,omitempty"`
	OrganizationId       int              `json:"organization_id"`
	OrganizationName     string           `json:"organization_name"`
	Title                string           `json:"title"`
	Subtitle             *string          `json:"subtitle,omitempty"`
	Description          string           `json:"description,omitempty"`
	Summary              *string          `json:"summary,omitempty"`
	EventTypes           []EventType      `json:"event_types,omitempty"`
	EventLinks           []WebLink        `json:"event_links,omitempty"`
	Tags                 []string         `json:"tags,omitempty"`
	OccasionType         *int             `json:"occasion_type,omitempty"`
	VenueId              *int             `json:"venue_id,omitempty"`
	VenueName            *string          `json:"venue_name,omitempty"`
	VenueStreet          *string          `json:"venue_street,omitempty"`
	VenueHouseNumber     *string          `json:"venue_house_number,omitempty"`
	VenuePostalCode      *string          `json:"venue_postal_code,omitempty"`
	VenueCity            *string          `json:"venue_city,omitempty"`
	VenueCountry         *string          `json:"venue_country,omitempty"`
	VenueState           *string          `json:"venue_state,omitempty"`
	VenueLon             *float64         `json:"venue_lon,omitempty"`
	VenueLat             *float64         `json:"venue_lat,omitempty"`
	SpaceId              *int             `json:"space_id,omitempty"`
	SpaceName            *string          `json:"space_name,omitempty"`
	SpaceTotalCapacity   *int             `json:"space_total_capacity,omitempty"`
	SpaceSeatingCapacity *int             `json:"space_seating_capacity,omitempty"`
	SpaceBuildingLevel   *int             `json:"space_building_level,omitempty"`
	OnlineLink           *string          `json:"online_link,omitempty"`
	MeetingPoint         *string          `json:"meeting_point,omitempty"`
	Languages            []string         `json:"languages,omitempty"`
	ParticipationInfo    *string          `json:"participation_info,omitempty"`
	MinAge               *int             `json:"min_age,omitempty"`
	MaxAge               *int             `json:"max_age,omitempty"`
	MaxAttendees         *int             `json:"max_attendees,omitempty"`
	PriceType            PriceType        `json:"price_type,omitempty"`
	MinPrice             *float64         `json:"min_price,omitempty"`
	MaxPrice             *float64         `json:"max_price,omitempty"`
	TicketFlags          []string         `json:"ticket_flags,omitempty"`
	Currency             *string          `json:"currency,omitempty"`
	CurrencyName         *string          `json:"currency_name,omitempty"`
	Images               []Image          `json:"images,omitempty"`
	Custom               *string          `json:"custom,omitempty"`
	Style                *string          `json:"style,omitempty"`
	EventDates           []AdminEventDate `json:"dates,omitempty"`
}

type AdminEventDate struct {
	Id                   int      `json:"id"`
	EventId              int      `json:"event_id"`
	StartDate            *string  `json:"start_date,omitempty"`
	StartTime            *string  `json:"start_time,omitempty"`
	EndDate              *string  `json:"end_date,omitempty"`
	EndTime              *string  `json:"end_time,omitempty"`
	EntryTime            *string  `json:"entry_time,omitempty"`
	Duration             *int64   `json:"duration,omitempty"`
	AllDay               *bool    `json:"all_day,omitempty"`
	AccessibilityInfo    *string  `json:"accessibility_info,omitempty"`
	VenueId              *int     `json:"venue_id,omitempty"`
	VenueName            *string  `json:"venue_name,omitempty"`
	VenueStreet          *string  `json:"venue_street,omitempty"`
	VenueHouseNumber     *string  `json:"venue_house_number,omitempty"`
	VenuePostalCode      *string  `json:"venue_postal_code,omitempty"`
	VenueCity            *string  `json:"venue_city,omitempty"`
	VenueState           *string  `json:"venue_state,omitempty"`
	VenueCountry         *string  `json:"venue_country,omitempty"`
	VenueLon             *float64 `json:"venue_lon,omitempty"`
	VenueLat             *float64 `json:"venue_lat,omitempty"`
	VenueLink            *string  `json:"venue_link,omitempty"`
	SpaceId              *int     `json:"space_id,omitempty"`
	SpaceName            *string  `json:"space_name,omitempty"`
	SpaceTotalCapacity   *int     `json:"space_total_capacity,omitempty"`
	SpaceSeatingCapacity *int     `json:"space_seating_capacity,omitempty"`
	SpaceBuildingLevel   *int     `json:"space_building_level,omitempty"`
	SpaceLink            *string  `json:"space_link,omitempty"`
}

type AdminListEvent struct {
	Id               int         `json:"id"`
	DateId           int         `json:"date_id"`
	ReleaseStatus    *string     `json:"release_status"`
	ReleaseDate      *string     `json:"release_date,omitempty"`
	CanEditEvent     bool        `json:"can_edit_event"`
	CanDeleteEvent   bool        `json:"can_delete_event"`
	CanReleaseEvent  bool        `json:"can_release_event"`
	OrganizationId   int         `json:"organization_id"`
	OrganizationName *string     `json:"organization_name"`
	VenueId          *int        `json:"venue_id,omitempty"`
	VenueName        *string     `json:"venue_name,omitempty"`
	SpaceId          *int        `json:"space_id,omitempty"`
	SpaceName        *string     `json:"space_name,omitempty"`
	ImageId          *int        `json:"image_id,omitempty"`
	ImageUrl         *string     `json:"image_url,omitempty"`
	SeriesIndex      int         `json:"series_index,omitempty"`
	SeriesTotal      int         `json:"series_total,omitempty"`
	StartDate        *string     `json:"start_date"`
	StartTime        *string     `json:"start_time"`
	EndDate          *string     `json:"end_date,omitempty"`
	EndTime          *string     `json:"end_time,omitempty"`
	Title            string      `json:"title"`
	Subtitle         *string     `json:"subtitle,omitempty"`
	EventTypes       []EventType `json:"event_types,omitempty"`
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
	TicketLink           *string `json:"ticket_link,omitempty"`
	AvailabilityStatusId *int    `json:"availability_status_id,omitempty"`
	AccessibilityInfo    *string `json:"accessibility_info,omitempty"`
	Custom               *string `json:"custom,omitempty"`
}

// UserEventNotification contains a single event notification
type UserEventNotification struct {
	EventId           int        `json:"event_id"`
	EventTitle        string     `json:"event_title"`
	OrganizationId    int        `json:"organization_id"`
	OrganizationName  *string    `json:"organization_name,omitempty"`
	ReleaseDate       *time.Time `json:"release_date,omitempty"`
	ReleaseStatus     string     `json:"release_status"`
	EarliestEventDate *time.Time `json:"earliest_event_date,omitempty"`
	LatestEventDate   *time.Time `json:"latest_event_date,omitempty"`
	DaysUntilRelease  *int       `json:"days_until_release,omitempty"`
	DaysUntilEvent    *int       `json:"days_until_event,omitempty"`
}
