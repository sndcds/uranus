package model

import (
	"time"
)

/*
type TicketFlag string

const (
	AdvanceTicket        TicketFlag = "advance_ticket"
	TicketRequired       TicketFlag = "ticket_required"
	OnSiteTicketSales    TicketFlag = "on_site_ticket_sales"
	RegistrationRequired TicketFlag = "registration_required"
)
*/

type PriceType string

const (
	NotSpecified PriceType = "not_specified"
	RegularPrice PriceType = "regular_price"
	Free         PriceType = "free"
	Donation     PriceType = "donation"
	TieredPrices PriceType = "tiered_prices"
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
	Uuid                         string   `json:"uuid"`
	EventId                      int      `json:"event_id"`
	EventReleaseStatus           string   `json:"release_status"`
	StartDate                    string   `json:"start_date"`
	StartTime                    string   `json:"start_time"`
	EndDate                      *string  `json:"end_date,omitempty"`
	EndTime                      *string  `json:"end_time,omitempty"`
	EntryTime                    *string  `json:"entry_time,omitempty"`
	Duration                     *int     `json:"duration,omitempty"`
	VenueId                      *int     `json:"venue_id"`
	VenueName                    *string  `json:"venue_name,omitempty"`
	VenueStreet                  *string  `json:"venue_street,omitempty"`
	VenueHouseNumber             *string  `json:"venue_house_number,omitempty"`
	VenuePostalCode              *string  `json:"venue_postal_code,omitempty"`
	VenueCity                    *string  `json:"venue_city,omitempty"`
	VenueCountry                 *string  `json:"venue_country,omitempty"`
	VenueState                   *string  `json:"venue_state,omitempty"`
	VenueLon                     *float32 `json:"venue_lon,omitempty"`
	VenueLat                     *float32 `json:"venue_lat,omitempty"`
	VenueWebLink                 *string  `json:"venue_web_link,omitempty"`
	VenueLogoImageUuid           *string  `json:"venue_logo_uuid,omitempty"`
	VenueLightThemeLogoImageUuid *string  `json:"venue_light_theme_logo_uuid,omitempty"`
	VenueDarkThemeLogoImageUuid  *string  `json:"venue_dark_theme_logo_uuid,omitempty"`
	VenueLogoUrl                 *string  `json:"venue_logo_url,omitempty"`
	VenueLightThemeLogoUrl       *string  `json:"venue_light_theme_logo_url,omitempty"`
	VenueDarkThemeLogoUrl        *string  `json:"venue_dark_theme_logo_url,omitempty"`
	SpaceId                      *int     `json:"space_id,omitempty"`
	SpaceName                    *string  `json:"space_name,omitempty"`
	SpaceWebLink                 *string  `json:"space_web_link,omitempty"`
	TotalCapacity                *int     `json:"total_capacity,omitempty"`
	SeatingCapacity              *int     `json:"seating_capacity,omitempty"`
	BuildingLevel                *int     `json:"building_level,omitempty"`
	AccessibilityFlags           *string  `json:"accessibility_flags,omitempty"`
	AccessibilitySummary         *string  `json:"accessibility_summary,omitempty"`
	AccessibilityInfo            *string  `json:"accessibility_info,omitempty"`
}

type EventDetails struct {
	Id                int         `json:"id"`
	ReleaseStatus     *string     `json:"release_status,omitempty"`
	Title             string      `json:"title"`
	Subtitle          *string     `json:"subtitle,omitempty"`
	Description       *string     `json:"description,omitempty"`
	Summary           *string     `json:"summary,omitempty"`
	Languages         []string    `json:"languages,omitempty"`
	Tags              []string    `json:"tags,omitempty"`
	OrganizationId    int         `json:"organization_id"`
	OrganizationName  string      `json:"organization_name"`
	OrganizationUrl   *string     `json:"organization_website,omitempty"`
	Image             *Image      `json:"image,omitempty"`       // nested struct for the JSON image
	EventTypes        []EventType `json:"event_types,omitempty"` // typed slice
	EventLinks        []WebLink   `json:"event_links,omitempty"` // typed slice
	Date              *EventDate  `json:"date,omitempty"`
	FurtherDates      []EventDate `json:"further_dates,omitempty"`
	MaxAttendees      *int        `json:"max_attendees,omitempty"`
	MinAge            *int        `json:"min_age,omitempty"`
	MaxAge            *int        `json:"max_age,omitempty"`
	Currency          *string     `json:"currency,omitempty"`
	PriceType         *string     `json:"price_type,omitempty"`
	MinPrice          *float64    `json:"min_price,omitempty"`
	MaxPrice          *float64    `json:"max_price,omitempty"`
	VisitorInfoFlags  *string     `json:"visitor_info_flags,omitempty"`
	ParticipationInfo *string     `json:"participation_info,omitempty"`
	MeetingPoint      *string     `json:"meeting_point,omitempty"`
}

type AdminEvent struct {
	Uuid                 string           `json:"id"`
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
	VisitorInfoFlags     *string          `json:"visitor_info_flags,omitempty"`
	Images               []Image          `json:"images,omitempty"`
	Custom               *string          `json:"custom,omitempty"`
	Style                *string          `json:"style,omitempty"`
	EventDates           []AdminEventDate `json:"dates,omitempty"`
}

type AdminEventDate struct {
	Uuid                 string   `json:"uuid"`
	EventUuid            string   `json:"event_uuid"`
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
	Uuid            string      `json:"uuid"`
	DateUuid        *string     `json:"date_uuid"`
	ReleaseStatus   *string     `json:"release_status"`
	ReleaseDate     *string     `json:"release_date,omitempty"`
	Categories      *[]int      `json:"categories,omitempty"`
	CanEditEvent    bool        `json:"can_edit_event"`
	CanDeleteEvent  bool        `json:"can_delete_event"`
	CanReleaseEvent bool        `json:"can_release_event"`
	OrgUuid         string      `json:"org_uuid"`
	OrgName         *string     `json:"org_name"`
	VenueUuid       *string     `json:"venue_uuid,omitempty"`
	VenueName       *string     `json:"venue_name,omitempty"`
	SpaceUuid       *string     `json:"space_uuid,omitempty"`
	SpaceName       *string     `json:"space_name,omitempty"`
	ImageId         *int        `json:"image_id,omitempty"`
	ImageUrl        *string     `json:"image_url,omitempty"`
	SeriesIndex     int         `json:"series_index,omitempty"`
	SeriesTotal     int         `json:"series_total,omitempty"`
	StartDate       *string     `json:"start_date"`
	StartTime       *string     `json:"start_time"`
	EndDate         *string     `json:"end_date,omitempty"`
	EndTime         *string     `json:"end_time,omitempty"`
	Title           string      `json:"title"`
	Subtitle        *string     `json:"subtitle,omitempty"`
	EventTypes      []EventType `json:"event_types,omitempty"`
}

// UserEventNotification contains a single event notification
type UserEventNotification struct {
	EventUuid         string     `json:"event_uuid"`
	EventTitle        string     `json:"event_title"`
	OrgUuid           string     `json:"org_uuid"`
	OrgName           *string    `json:"org_name,omitempty"`
	ReleaseDate       *time.Time `json:"release_date,omitempty"`
	ReleaseStatus     string     `json:"release_status"`
	EarliestEventDate *time.Time `json:"earliest_event_date,omitempty"`
	LatestEventDate   *time.Time `json:"latest_event_date,omitempty"`
	DaysUntilRelease  *int       `json:"days_until_release,omitempty"`
	DaysUntilEvent    *int       `json:"days_until_event,omitempty"`
}
