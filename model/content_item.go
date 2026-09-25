package model

// ContentItem is a runtime view of a source, independent of posts and targets.
// Text is kept verbatim; Language applies to labels, ContentLanguage to source text.
type ContentItem struct {
	SourceType      string           `json:"source_type"`
	SourceUuid      string           `json:"source_uuid"`
	OrgUuid         string           `json:"org_uuid"`
	Language        string           `json:"language"`
	ContentLanguage *string          `json:"content_language,omitempty"`
	Title           string           `json:"title"`
	Subtitle        *string          `json:"subtitle,omitempty"`
	Description     *string          `json:"description,omitempty"`
	Summary         *string          `json:"summary,omitempty"`
	Url             *string          `json:"url,omitempty"`
	TicketLink      *string          `json:"ticket_link,omitempty"`
	OnlineLink      *string          `json:"online_link,omitempty"`
	Location        *ContentLocation `json:"location,omitempty"`
	Dates           []ContentDate    `json:"dates"`
	Images          []Image          `json:"images"`
}

// ContentLocation contains the existing venue or organization address.
type ContentLocation struct {
	Uuid        string  `json:"uuid"`
	Name        string  `json:"name"`
	Street      *string `json:"street,omitempty"`
	HouseNumber *string `json:"house_number,omitempty"`
	PostalCode  *string `json:"postal_code,omitempty"`
	City        *string `json:"city,omitempty"`
	State       *string `json:"state,omitempty"`
	Country     *string `json:"country,omitempty"`
	Url         *string `json:"url,omitempty"`
}

// ContentDate keeps local date/time values separate, including missing times.
// No timezone, end time, or preferred occurrence is inferred.
type ContentDate struct {
	Uuid       string           `json:"uuid"`
	StartDate  string           `json:"start_date"`
	StartTime  *string          `json:"start_time,omitempty"`
	EndDate    *string          `json:"end_date,omitempty"`
	EndTime    *string          `json:"end_time,omitempty"`
	EntryTime  *string          `json:"entry_time,omitempty"`
	Duration   *int             `json:"duration,omitempty"`
	AllDay     *bool            `json:"all_day,omitempty"`
	TicketLink *string          `json:"ticket_link,omitempty"`
	Location   *ContentLocation `json:"location,omitempty"`
	SpaceUuid  *string          `json:"space_uuid,omitempty"`
	SpaceName  *string          `json:"space_name,omitempty"`
}
