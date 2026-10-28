package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/sndcds/uranus/model"
)

const maxICSSize = 20 << 20 // 20 MiB

func ReadICSURL(
	ctx context.Context,
	feedURL string,
	sourceID string,
) ([]model.ImportedEvent, error) {
	parsedURL, err := url.ParseRequestURI(feedURL)
	if err != nil {
		return nil, fmt.Errorf("invalid ICS URL: %w", err)
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return nil, fmt.Errorf("unsupported ICS URL scheme: %q", parsedURL.Scheme)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		feedURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create ICS request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download ICS feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("ICS feed returned HTTP status %s", resp.Status)
	}

	data, err := io.ReadAll(
		io.LimitReader(resp.Body, maxICSSize+1),
	)
	if err != nil {
		return nil, fmt.Errorf("read ICS feed: %w", err)
	}

	if len(data) > maxICSSize {
		return nil, fmt.Errorf("ICS feed exceeds %d bytes", maxICSSize)
	}

	calendar, err := ics.ParseCalendar(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse ICS calendar: %w", err)
	}

	calendarEvents := calendar.Events()
	imported := make([]model.ImportedEvent, 0, len(calendarEvents))

	for index, event := range calendarEvents {
		item, err := convertICSEvent(event, sourceID)
		if err != nil {
			return nil, fmt.Errorf("VEVENT %d: %w", index+1, err)
		}

		imported = append(imported, item)
	}

	return imported, nil
}

func convertICSEvent(
	event *ics.VEvent,
	sourceID string,
) (model.ImportedEvent, error) {
	uid := icalValue(event, "UID")
	if uid == "" {
		return model.ImportedEvent{}, fmt.Errorf("missing UID")
	}

	title := icalValue(event, "SUMMARY")
	if title == "" {
		return model.ImportedEvent{}, fmt.Errorf("missing SUMMARY for UID %q", uid)
	}

	startProperty := event.GetProperty(ics.ComponentPropertyDtStart)
	if startProperty == nil {
		return model.ImportedEvent{}, fmt.Errorf("missing DTSTART for UID %q", uid)
	}

	startsAt, err := event.GetStartAt()
	if err != nil {
		return model.ImportedEvent{}, fmt.Errorf(
			"parse DTSTART for UID %q: %w", uid, err,
		)
	}

	allDay := isAllDay(startProperty)

	endsAt, err := event.GetEndAt()
	if err != nil {
		// RFC 5545 defaults: one day for a DATE event,
		// zero duration for a DATE-TIME event without DTEND.
		endsAt = startsAt
		if allDay {
			endsAt = startsAt.AddDate(0, 0, 1)
		}
	}

	var categories []string
	for _, property := range event.GetProperties(
		ics.ComponentPropertyCategories,
	) {
		for _, category := range strings.Split(property.Value, ",") {
			category = strings.TrimSpace(category)
			if category != "" {
				categories = append(categories, category)
			}
		}
	}

	var recurrenceRules []string
	for _, property := range event.GetProperties(
		ics.ComponentPropertyRrule,
	) {
		recurrenceRules = append(recurrenceRules, property.Value)
	}

	return model.ImportedEvent{
		SourceID:        sourceID,
		ExternalUID:     uid,
		Title:           title,
		Description:     icalValue(event, "DESCRIPTION"),
		DescriptionHTML: icalValue(event, "X-ALT-DESC"),
		SourceURL:       icalValue(event, "URL"),
		ImageURL:        icalValue(event, "IMAGE"),
		LocationText:    icalValue(event, "LOCATION"),
		Categories:      categories,
		Status:          icalValue(event, "STATUS"),
		StartsAt:        startsAt,
		EndsAt:          endsAt,
		AllDay:          allDay,
		RecurrenceRules: recurrenceRules,
	}, nil
}

func icalValue(event *ics.VEvent, name string) string {
	property := event.GetProperty(ics.ComponentProperty(name))
	if property == nil {
		return ""
	}

	return strings.TrimSpace(property.Value)
}

func isAllDay(property *ics.IANAProperty) bool {
	for key, values := range property.ICalParameters {
		if !strings.EqualFold(key, "VALUE") {
			continue
		}

		for _, value := range values {
			if strings.EqualFold(value, "DATE") {
				return true
			}
		}
	}

	return false
}
