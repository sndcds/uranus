package service

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sndcds/uranus/model"
)

// Version 1 thresholds are editorial heuristics, not publication requirements.
const (
	titleMin                 = 10
	titleMax                 = 120
	descriptionMin           = 200
	descriptionMax           = 10000
	descriptionMinWords      = 40
	descriptionMaxParagraphs = 20
	descriptionMaxHeadings   = 8
	summaryMin               = 80
	summaryMax               = 500
	summaryMaxEmphasis       = 3
	summaryMaxEmphasisRatio  = 0.25
	imageMinLongEdge         = 1200
	imageMinShortEdge        = 630
)

type qualityEvaluator struct{ report model.EventQualityReport }

// EvaluateEventQuality inspects persisted event data without I/O or mutations.
// Completeness measures presence only; issues separately describe quality.
func EvaluateEventQuality(event model.AdminEvent) model.EventQualityReport {
	q := qualityEvaluator{report: model.EventQualityReport{
		EventUuid: event.Uuid, RulesVersion: 1, Status: "ok",
		Issues: make([]model.EventQualityIssue, 0),
	}}
	title := utf8.RuneCountInString(strings.Join(strings.Fields(event.Title), " "))
	description := analyzeEventText(value(event.Description))
	summary := analyzeEventText(value(event.Summary))
	q.report.Metrics = model.EventQualityMetrics{
		TitleCharacters: title, Description: description, Summary: summary,
		ImageCount: len(event.Images), DateCount: len(event.EventDates),
	}
	q.presence(title > 0, "title", "title.missing", "Ein Titel fehlt.", "Einen aussagekräftigen Titel ergänzen.", "")
	if title > 0 && (title < titleMin || title > titleMax) {
		q.add("title.length", "recommendation", "title", "Der Titel ist sehr kurz oder sehr lang.", "Den Titel auf 10 bis 120 Zeichen begrenzen und die Veranstaltung klar benennen.", "", "", map[string]any{"actual": title, "min": titleMin, "max": titleMax})
	}
	q.presence(description.Characters > 0, "description", "description.missing", "Eine Beschreibung fehlt.", "Inhalt, Ablauf und Besonderheiten der Veranstaltung beschreiben.", "")
	if description.Characters > 0 {
		if description.Characters < descriptionMin || description.Words < descriptionMinWords {
			q.add("description.too_short", "recommendation", "description", "Die Beschreibung ist sehr kurz.", "Mindestens 200 Zeichen und etwa 40 Wörter mit hilfreichen Informationen ergänzen.", "", "", nil)
		}
		if description.Sentences < 2 {
			q.add("description.single_sentence", "recommendation", "description", "Die Beschreibung besteht aus höchstens einem erkennbaren Satz.", "Prüfen, ob Inhalt, Ablauf und Zielgruppe ausreichend erklärt sind.", "", "", nil)
		}
		if description.Characters > descriptionMax {
			q.add("description.too_long", "recommendation", "description", "Die Beschreibung ist sehr lang.", "Auf höchstens 10.000 Zeichen kürzen und Wiederholungen entfernen.", "", "", nil)
		}
		if description.Paragraphs > descriptionMaxParagraphs {
			q.add("description.too_many_paragraphs", "recommendation", "description", "Die Beschreibung enthält sehr viele Textblöcke.", "Verwandte Inhalte zusammenfassen; höchstens 20 Textblöcke anstreben.", "", "", nil)
		}
		if description.Headings > descriptionMaxHeadings {
			q.add("description.too_many_headings", "recommendation", "description", "Die Beschreibung enthält sehr viele Überschriften.", "Die Gliederung auf höchstens 8 Überschriften reduzieren.", "", "", nil)
		}
	}
	q.presence(summary.Characters > 0, "summary", "summary.missing", "Eine Zusammenfassung fehlt.", "Eine kurze Zusammenfassung als zusammenhängenden Text ergänzen.", "")
	if summary.Characters > 0 {
		if summary.Characters < summaryMin || summary.Characters > summaryMax {
			q.add("summary.length", "recommendation", "summary", "Die Zusammenfassung ist sehr kurz oder sehr lang.", "Die Kernaussage in 80 bis 500 Zeichen zusammenfassen.", "", "", map[string]any{"actual": summary.Characters, "min": summaryMin, "max": summaryMax})
		}
		if summary.Paragraphs > 1 {
			q.add("summary.too_many_paragraphs", "recommendation", "summary", "Die Zusammenfassung enthält mehrere Textblöcke oder Zeilenumbrüche.", "Einen zusammenhängenden Text ohne Absatzumbrüche verwenden.", "", "", nil)
		}
		if summary.EmphasisCount > summaryMaxEmphasis || summary.EmphasisRatio > summaryMaxEmphasisRatio {
			q.add("summary.too_much_emphasis", "recommendation", "summary", "Die Zusammenfassung enthält viele Hervorhebungen.", "Fett und kursiv sparsam einsetzen: höchstens 3 Hervorhebungen und 25 Prozent des Textes.", "", "", nil)
		}
		if summary.OtherFormatting > 0 {
			q.add("summary.formatting", "recommendation", "summary", "Die Zusammenfassung enthält zusätzliche Formatierungen.", "Überschriften, Listen, Links, Bilder, eigene Stile und andere Formatierungen entfernen; einfacher Text mit sparsamem Fett/Kursiv genügt.", "", "", nil)
		}
	}
	q.presence(len(event.Images) > 0, "images", "image.missing", "Ein Veranstaltungsbild fehlt.", "Ein passendes Bild mit mindestens 1200 Pixeln an der langen und 630 Pixeln an der kurzen Seite ergänzen.", "")
	for _, img := range event.Images {
		if img.Width == nil || img.Height == nil || *img.Width <= 0 || *img.Height <= 0 {
			q.add("image.resolution_unknown", "info", "images", "Die Bildauflösung ist nicht bekannt.", "Die Bildmaße in den Metadaten ergänzen oder das Bild erneut hochladen.", "", img.Uuid, map[string]any{"identifier": img.Identifier})
			continue
		}
		long, short := max(*img.Width, *img.Height), min(*img.Width, *img.Height)
		if long < imageMinLongEdge || short < imageMinShortEdge {
			q.add("image.low_resolution", "warning", "images", "Das Bild hat eine geringe Auflösung.", "Ein Bild mit mindestens 1200 Pixeln an der langen und 630 Pixeln an der kurzen Seite verwenden.", "", img.Uuid, map[string]any{"identifier": img.Identifier, "width": *img.Width, "height": *img.Height, "min_long_edge": imageMinLongEdge, "min_short_edge": imageMinShortEdge})
		}
	}
	q.presence(len(event.EventTypes) > 0, "event_types", "event_types.missing", "Eine Veranstaltungsart fehlt.", "Mindestens eine passende Veranstaltungsart auswählen.", "")
	q.presence(len(event.EventDates) > 0, "dates", "dates.missing", "Es sind keine Termine vorhanden.", "Mindestens einen Veranstaltungstermin ergänzen.", "")
	for _, date := range event.EventDates {
		q.evaluateDate(event, date)
	}
	c := &q.report.Completeness
	c.Score = int(math.Round(100 * float64(c.Present) / float64(c.Total)))
	return q.report
}

func (q *qualityEvaluator) presence(present bool, field, code, message, recommendation, dateUuid string) {
	q.report.Completeness.Total++
	if present {
		q.report.Completeness.Present++
		return
	}
	q.add(code, "warning", field, message, recommendation, dateUuid, "", nil)
}

func (q *qualityEvaluator) add(code, severity, field, message, recommendation, dateUuid, imageUuid string, details map[string]any) {
	q.report.Issues = append(q.report.Issues, model.EventQualityIssue{
		Code: code, Severity: severity, Field: field, Message: message,
		Recommendation: recommendation, EventDateUuid: dateUuid, ImageUuid: imageUuid, Details: details,
	})
	if severity == "warning" {
		q.report.Status = "warnings"
	} else if q.report.Status == "ok" {
		q.report.Status = "recommendations"
	}
}

func (q *qualityEvaluator) evaluateDate(event model.AdminEvent, date model.AdminEventDate) {
	id := date.Uuid
	allDay := date.AllDay != nil && *date.AllDay
	q.presence(value(date.StartDate) != "", "dates.start_date", "date.start_date_missing", "Das Startdatum fehlt.", "Das Datum des Termins ergänzen.", id)
	if !allDay {
		q.presence(value(date.StartTime) != "", "dates.start_time", "date.start_time_missing", "Die Startzeit fehlt.", "Eine Startzeit angeben oder den Termin als ganztägig kennzeichnen.", id)
	}
	q.presence(value(date.VenueUuid) != "" || value(event.VenueUuid) != "" || value(event.OnlineLink) != "", "dates.venue_uuid", "date.location_missing", "Für den Termin fehlt ein Veranstaltungsort oder Online-Link.", "Am Termin oder Event einen Veranstaltungsort oder am Event einen Online-Link ergänzen.", id)

	parsed := make(map[string]time.Time)
	valid := make(map[string]bool)
	for _, field := range []struct {
		name   string
		value  *string
		layout string
	}{
		{"start_date", date.StartDate, "2006-01-02"}, {"end_date", date.EndDate, "2006-01-02"},
		{"start_time", date.StartTime, "15:04"}, {"end_time", date.EndTime, "15:04"}, {"entry_time", date.EntryTime, "15:04"},
	} {
		if value(field.value) == "" {
			continue
		}
		t, err := time.Parse(field.layout, value(field.value))
		if err != nil {
			q.add("date."+field.name+"_invalid", "warning", "dates."+field.name, "Der Termin enthält ein ungültiges Datum oder eine ungültige Uhrzeit.", fmt.Sprintf("Das Feld %s prüfen (Format %s).", field.name, field.layout), id, "", nil)
		} else {
			parsed[field.name], valid[field.name] = t, true
		}
	}
	if valid["start_date"] && valid["end_date"] && parsed["end_date"].Before(parsed["start_date"]) {
		q.add("date.end_before_start", "warning", "dates.end_date", "Das Enddatum liegt vor dem Startdatum.", "Start- und Enddatum korrigieren.", id, "", nil)
	} else if !allDay && valid["start_date"] && valid["start_time"] && valid["end_time"] && (value(date.EndDate) == "" || valid["end_date"] && parsed["end_date"].Equal(parsed["start_date"])) && !parsed["end_time"].After(parsed["start_time"]) {
		q.add("date.end_before_start", "warning", "dates.end_time", "Die Endzeit liegt am selben Tag nicht nach der Startzeit.", "Die Zeiten prüfen; bei einem Termin über Mitternacht das Enddatum auf den Folgetag setzen.", id, "", nil)
	}
	if date.Duration != nil && *date.Duration <= 0 {
		q.add("date.duration_invalid", "warning", "dates.duration", "Die Dauer ist nicht positiv.", "Eine positive Dauer angeben oder das Feld leeren.", id, "", nil)
	}
	if !allDay && value(date.EndTime) == "" && date.Duration == nil {
		q.add("date.end_missing", "info", "dates.end_time", "Endzeit und Dauer sind nicht angegeben.", "Wenn bekannt, eine Endzeit oder Dauer ergänzen, damit Gäste besser planen können.", id, "", nil)
	}
}

func value(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
