package model

import "time"

type EventTicketFlag string

const (
	EventAdvanceTicket         EventTicketFlag = "advance_ticket"
	EventPresaleFeeApplies     EventTicketFlag = "presale_fee_applies"
	EventSiteTicketSales       EventTicketFlag = "on_site_ticket_sales"
	EventReducedPriceAvailable EventTicketFlag = "reduced_price_available"
)

type EventPriceType string

const (
	NotSpecified EventPriceType = "not_specified"
	RegularPrice EventPriceType = "regular_price"
	Free         EventPriceType = "free"
	Donation     EventPriceType = "donation"
	TieredPrices EventPriceType = "tiered_prices"
)

type EventTypeGenrePairPayload struct {
	TypeId  int  `json:"type_id" binding:"required"`
	GenreId *int `json:"genre_id"`
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
	VenueUuid            *string `json:"venue_uuid,omitempty"`
	SpaceUuid            *string `json:"space_uuid,omitempty"`
	TicketLink           *string `json:"ticket_link,omitempty"`
	AvailabilityStatusId *int    `json:"availability_status_id,omitempty"`
	AccessibilityInfo    *string `json:"accessibility_info,omitempty"`
	Custom               *string `json:"custom,omitempty"`
}

type EventType struct {
	Type      int     `json:"type_id"`
	TypeName  *string `json:"type_name,omitempty"`
	Genre     int     `json:"genre_id"`
	GenreName *string `json:"genre_name,omitempty"`
}

type VenueInfo struct {
	VenueUuid string  `json:"venue_uuid"`
	VenueName *string `json:"venue_name"`
	SpaceUuid *string `json:"space_uuid"`
	SpaceName *string `json:"space_name"`
	City      *string `json:"city"`
	Country   *string `json:"country"`
}

type EventDate struct {
	Uuid                 string          `json:"uuid"`
	Slug                 string          `json:"slug"`
	EventUuid            string          `json:"event_uuid"`
	EventReleaseStatus   string          `json:"release_status"`
	StartDate            string          `json:"start_date"`
	StartTime            string          `json:"start_time"`
	EndDate              *string         `json:"end_date,omitempty"`
	EndTime              *string         `json:"end_time,omitempty"`
	EntryTime            *string         `json:"entry_time,omitempty"`
	Duration             *int            `json:"duration,omitempty"`
	VenueUuid            *string         `json:"venue_uuid"`
	VenueName            *string         `json:"venue_name,omitempty"`
	VenueStreet          *string         `json:"venue_street,omitempty"`
	VenueHouseNumber     *string         `json:"venue_house_number,omitempty"`
	VenuePostalCode      *string         `json:"venue_postal_code,omitempty"`
	VenueCity            *string         `json:"venue_city,omitempty"`
	VenueCountry         *string         `json:"venue_country,omitempty"`
	VenueState           *string         `json:"venue_state,omitempty"`
	VenueLon             *float32        `json:"venue_lon,omitempty"`
	VenueLat             *float32        `json:"venue_lat,omitempty"`
	VenueWebLink         *string         `json:"venue_web_link,omitempty"`
	VenueLogos           map[string]Logo `json:"venue_logos"`
	SpaceUuid            *string         `json:"space_uuid,omitempty"`
	SpaceName            *string         `json:"space_name,omitempty"`
	SpaceWebLink         *string         `json:"space_web_link,omitempty"`
	TotalCapacity        *int            `json:"total_capacity,omitempty"`
	SeatingCapacity      *int            `json:"seating_capacity,omitempty"`
	BuildingLevel        *int            `json:"building_level,omitempty"`
	AccessibilityFlags   *string         `json:"accessibility_flags,omitempty"`
	AccessibilitySummary *string         `json:"accessibility_summary,omitempty"`
	AccessibilityInfo    *string         `json:"accessibility_info,omitempty"`
	AccessibilityLabels  []string        `json:"accessibility_labels,omitempty"`
}

type EventDetails struct {
	Uuid                 string           `json:"uuid"`
	ReleaseStatus        *string          `json:"release_status,omitempty"`
	ContentLanguage      *string          `json:"content_language,omitempty"`
	Title                string           `json:"title"`
	Subtitle             *string          `json:"subtitle,omitempty"`
	Description          *string          `json:"description,omitempty"`
	Summary              *string          `json:"summary,omitempty"`
	SourceUrl            *string          `json:"source_link,omitempty"`
	Languages            []string         `json:"languages,omitempty"`
	Tags                 []string         `json:"tags,omitempty"`
	OrgUuid              string           `json:"org_uuid"`
	OrgName              string           `json:"org_name"`
	OrgWebLink           *string          `json:"org_web_link,omitempty"`
	OrgLogos             map[string]Logo  `json:"org_logos"`
	Images               map[string]Image `json:"images,omitempty"`
	EventTypes           []EventType      `json:"event_types,omitempty"` // Typed slice
	EventLinks           []WebLink        `json:"event_links,omitempty"` // Typed slice
	Date                 *EventDate       `json:"date,omitempty"`
	FurtherDates         []EventDate      `json:"further_dates,omitempty"`
	MaxAttendees         *int             `json:"max_attendees,omitempty"`
	MinAge               *int             `json:"min_age,omitempty"`
	MaxAge               *int             `json:"max_age,omitempty"`
	Currency             *string          `json:"currency,omitempty"`
	PriceType            *string          `json:"price_type,omitempty"`
	MinPrice             *float64         `json:"min_price,omitempty"`
	MaxPrice             *float64         `json:"max_price,omitempty"`
	TicketFlags          []string         `json:"ticket_flags,omitempty"`
	TicketLink           *string          `json:"ticket_link,omitempty"`
	VisitorInfoFlags     *string          `json:"visitor_info_flags,omitempty"`
	ParticipationInfo    *string          `json:"participation_info,omitempty"`
	MeetingPoint         *string          `json:"meeting_point,omitempty"`
	OnlineLink           *string          `json:"online_link,omitempty"`
	RegistrationLink     *string          `json:"registration_link,omitempty"`
	RegistrationEmail    *string          `json:"registration_email,omitempty"`
	RegistrationPhone    *string          `json:"registration_phone,omitempty"`
	RegistrationDeadline *string          `json:"registration_deadline,omitempty"`
	LogoMode             int              `json:"logo_mode,omitempty"`
}

type AdminEvent struct {
	Uuid                 string           `json:"uuid"`
	ExternalId           *string          `json:"external_id,omitempty"`
	SourceLink           *string          `json:"source_link,omitempty"`
	ReleaseStatus        string           `json:"release_status"`
	ReleaseDate          *string          `json:"release_date,omitempty"`
	Categories           []int            `json:"categories,omitempty"`
	ContentLanguage      *string          `json:"content_language,omitempty"`
	OrgUuid              string           `json:"org_uuid"`
	OrgName              string           `json:"org_name"`
	Title                string           `json:"title"`
	Subtitle             *string          `json:"subtitle,omitempty"`
	Description          *string          `json:"description,omitempty"`
	Summary              *string          `json:"summary,omitempty"`
	EventTypes           []EventType      `json:"event_types,omitempty"`
	EventLinks           []WebLink        `json:"event_links,omitempty"`
	Tags                 []string         `json:"tags,omitempty"`
	OccasionType         *int             `json:"occasion_type,omitempty"`
	VenueUuid            *string          `json:"venue_uuid,omitempty"`
	VenueName            *string          `json:"venue_name,omitempty"`
	VenueStreet          *string          `json:"venue_street,omitempty"`
	VenueHouseNumber     *string          `json:"venue_house_number,omitempty"`
	VenuePostalCode      *string          `json:"venue_postal_code,omitempty"`
	VenueCity            *string          `json:"venue_city,omitempty"`
	VenueCountry         *string          `json:"venue_country,omitempty"`
	VenueState           *string          `json:"venue_state,omitempty"`
	VenueLon             *float64         `json:"venue_lon,omitempty"`
	VenueLat             *float64         `json:"venue_lat,omitempty"`
	SpaceUuid            *string          `json:"space_uuid,omitempty"`
	SpaceName            *string          `json:"space_name,omitempty"`
	SpaceTotalCapacity   *int             `json:"space_total_capacity,omitempty"`
	SpaceSeatingCapacity *int             `json:"space_seating_capacity,omitempty"`
	SpaceBuildingLevel   *int             `json:"space_building_level,omitempty"`
	OnlineLink           *string          `json:"online_link,omitempty"`
	RegistrationLink     *string          `json:"registration_link,omitempty"`
	RegistrationEmail    *string          `json:"registration_email,omitempty"`
	RegistrationPhone    *string          `json:"registration_phone,omitempty"`
	RegistrationDeadline *string          `json:"registration_deadline,omitempty"`
	MeetingPoint         *string          `json:"meeting_point,omitempty"`
	Languages            []string         `json:"languages,omitempty"`
	ParticipationInfo    *string          `json:"participation_info,omitempty"`
	MinAge               *int             `json:"min_age,omitempty"`
	MaxAge               *int             `json:"max_age,omitempty"`
	MaxAttendees         *int             `json:"max_attendees,omitempty"`
	PriceType            EventPriceType   `json:"price_type,omitempty"`
	MinPrice             *float64         `json:"min_price,omitempty"`
	MaxPrice             *float64         `json:"max_price,omitempty"`
	TicketFlags          []string         `json:"ticket_flags,omitempty"`
	TicketLink           *string          `json:"ticket_link,omitempty"`
	Currency             *string          `json:"currency,omitempty"`
	CurrencyName         *string          `json:"currency_name,omitempty"`
	VisitorInfoFlags     *string          `json:"visitor_info_flags,omitempty"`
	Images               []Image          `json:"images,omitempty"`
	Custom               *string          `json:"custom,omitempty"`
	Style                *string          `json:"style,omitempty"`
	EventDates           []AdminEventDate `json:"dates,omitempty"`
	CanRelease           bool             `json:"can_release,omitempty"`
}

type AdminEventDate struct {
	Uuid                 string   `json:"uuid"`
	EventUuid            string   `json:"event_uuid"`
	ReleaseStatus        *string  `json:"release_status"`
	StartDate            *string  `json:"start_date,omitempty"`
	StartTime            *string  `json:"start_time,omitempty"`
	EndDate              *string  `json:"end_date,omitempty"`
	EndTime              *string  `json:"end_time,omitempty"`
	EntryTime            *string  `json:"entry_time,omitempty"`
	Duration             *int64   `json:"duration,omitempty"`
	AllDay               *bool    `json:"all_day,omitempty"`
	AccessibilityInfo    *string  `json:"accessibility_info,omitempty"`
	VenueUuid            *string  `json:"venue_uuid,omitempty"`
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
	SpaceUuid            *string  `json:"space_uuid,omitempty"`
	SpaceName            *string  `json:"space_name,omitempty"`
	SpaceTotalCapacity   *int     `json:"space_total_capacity,omitempty"`
	SpaceSeatingCapacity *int     `json:"space_seating_capacity,omitempty"`
	SpaceBuildingLevel   *int     `json:"space_building_level,omitempty"`
	SpaceLink            *string  `json:"space_link,omitempty"`
}

type AdminListEvent struct {
	Uuid                 string      `json:"uuid"`
	DateUuid             *string     `json:"date_uuid"`
	ReleaseStatus        *string     `json:"release_status"`
	ReleaseDate          *string     `json:"release_date,omitempty"`
	Categories           *[]int      `json:"categories,omitempty"`
	EventTypes           []EventType `json:"event_types,omitempty"`
	CanEditEvent         bool        `json:"can_edit_event"`
	CanDeleteEvent       bool        `json:"can_delete_event"`
	CanReleaseEvent      bool        `json:"can_release_event"`
	CanViewEventInsights bool        `json:"can_view_event_insights"`
	OnlineLink           *string     `json:"online_link"`
	OrgUuid              string      `json:"org_uuid"`
	OrgName              *string     `json:"org_name"`
	OrgCity              *string     `json:"org_city"`
	VenueUuid            *string     `json:"venue_uuid,omitempty"`
	VenueName            *string     `json:"venue_name,omitempty"`
	VenueCity            *string     `json:"venue_city,omitempty"`
	SpaceUuid            *string     `json:"space_uuid,omitempty"`
	SpaceName            *string     `json:"space_name,omitempty"`
	ImageUuid            *string     `json:"image_uuid,omitempty"`
	ImageUrl             *string     `json:"image_url,omitempty"`
	SeriesIndex          int         `json:"series_index,omitempty"`
	SeriesTotal          int         `json:"series_total,omitempty"`
	StartDate            *string     `json:"start_date"`
	StartTime            *string     `json:"start_time"`
	EndDate              *string     `json:"end_date,omitempty"`
	EndTime              *string     `json:"end_time,omitempty"`
	Title                string      `json:"title"`
	Subtitle             *string     `json:"subtitle,omitempty"`
	UpcomingDatesCount   int         `json:"upcoming_dates_count,omitempty"`
	NextDate             *string     `json:"next_date,omitempty"`
}

// UserEventNotification contains a single event notification
type UserEventNotification struct {
	// Event core
	EventUuid  string `db:"uuid" json:"event_uuid"`
	EventTitle string `db:"title" json:"event_title"`
	OrgUuid    string `db:"org_uuid" json:"org_uuid"`
	OrgName    string `db:"org_name" json:"org_name"`

	// Venue (nullable)
	VenueUuid *string `db:"venue_uuid" json:"venue_uuid"`
	VenueName *string `db:"venue_name" json:"venue_name"`
	VenueCity *string `db:"venue_city" json:"venue_city"`

	// Release / schedule
	ReleaseStatus      *string    `db:"release_status" json:"release_status"`
	FirstDate          *time.Time `db:"first_date" json:"first_date"`
	DaysUntilFirstDate *int       `db:"days_until_first_date" json:"days_until_first_date"`

	// QA flags
	NoImage             bool `db:"no_image" json:"no_image"`
	NoEventDates        bool `db:"no_event_dates" json:"no_event_dates"`
	NoVenueOrOnlineLink bool `db:"no_venue_or_online_link" json:"no_venue_or_online_link"`
	NoEventType         bool `db:"no_event_type" json:"no_event_type"`
	NoTitle             bool `db:"no_title" json:"no_title"`
	NoUpcomingDate      bool `db:"no_upcoming_date" json:"no_upcoming_date"`
}
