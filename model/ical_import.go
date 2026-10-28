package model

import "time"

type ImportedEvent struct {
	SourceID        string
	ExternalUID     string
	Title           string
	Description     string
	DescriptionHTML string
	SourceURL       string
	ImageURL        string
	LocationText    string
	Categories      []string
	Status          string

	StartsAt time.Time
	EndsAt   time.Time
	AllDay   bool

	RecurrenceRules []string
}
