package model

// EventQualityReport is advisory and never changes publication status.
type EventQualityReport struct {
	EventUuid    string                   `json:"event_uuid"`
	RulesVersion int                      `json:"rules_version"`
	Status       string                   `json:"status"`
	Completeness EventQualityCompleteness `json:"completeness"`
	Metrics      EventQualityMetrics      `json:"metrics"`
	Issues       []EventQualityIssue      `json:"issues"`
}

type EventQualityCompleteness struct {
	Score   int `json:"score"`
	Present int `json:"present"`
	Total   int `json:"total"`
}

type EventQualityIssue struct {
	Code           string         `json:"code"`
	Severity       string         `json:"severity"`
	Field          string         `json:"field"`
	EventDateUuid  string         `json:"event_date_uuid,omitempty"`
	ImageUuid      string         `json:"image_uuid,omitempty"`
	Message        string         `json:"message"`
	Recommendation string         `json:"recommendation"`
	Details        map[string]any `json:"details,omitempty"`
}

type EventQualityMetrics struct {
	TitleCharacters int              `json:"title_characters"`
	Description     EventTextMetrics `json:"description"`
	Summary         EventTextMetrics `json:"summary"`
	ImageCount      int              `json:"image_count"`
	DateCount       int              `json:"date_count"`
}

type EventTextMetrics struct {
	Characters      int     `json:"characters"`
	Words           int     `json:"words"`
	Sentences       int     `json:"sentences"`
	Paragraphs      int     `json:"paragraphs"`
	Headings        int     `json:"headings"`
	EmphasisCount   int     `json:"emphasis_count"`
	EmphasisRatio   float64 `json:"emphasis_ratio"`
	OtherFormatting int     `json:"other_formatting"`
}
