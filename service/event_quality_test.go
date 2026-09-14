package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sndcds/uranus/model"
)

func qualityPtr[T any](v T) *T { return &v }

func completeQualityEvent() model.AdminEvent {
	return model.AdminEvent{
		Uuid: "event-1", Title: "Konzert im Kulturhaus",
		Description: qualityPtr(strings.Repeat("Die Gäste erleben Musik und können mit den Künstlern über ihre Arbeit sprechen. ", 4)),
		Summary:     qualityPtr("Ein Konzert im Kulturhaus mit Musik, persönlichen Begegnungen und einem abwechslungsreichen Programm für alle Gäste."),
		VenueUuid:   qualityPtr("venue-1"), EventTypes: []model.EventType{{Type: 1}},
		Images:     []model.Image{{Uuid: "image-1", Identifier: "main", Width: qualityPtr(int32(1200)), Height: qualityPtr(int32(630))}},
		EventDates: []model.AdminEventDate{{Uuid: "date-1", StartDate: qualityPtr("2026-10-10"), StartTime: qualityPtr("20:00"), EndTime: qualityPtr("22:00")}},
	}
}

func issueWithCode(report model.EventQualityReport, code string) *model.EventQualityIssue {
	for i := range report.Issues {
		if report.Issues[i].Code == code {
			return &report.Issues[i]
		}
	}
	return nil
}

func TestEventQualityComplete(t *testing.T) {
	report := EvaluateEventQuality(completeQualityEvent())
	if report.Status != "ok" || report.Completeness.Score != 100 || report.Completeness.Total != 9 || len(report.Issues) != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	encoded, err := json.Marshal(report)
	if err != nil || !strings.Contains(string(encoded), `"issues":[]`) {
		t.Fatalf("empty issues must be a JSON array: %s (%v)", encoded, err)
	}
}

func TestEventQualityEmpty(t *testing.T) {
	report := EvaluateEventQuality(model.AdminEvent{})
	if report.Status != "warnings" || report.Completeness.Score != 0 || report.Completeness.Total != 6 || len(report.Issues) != 6 {
		t.Fatalf("unexpected report: %+v", report)
	}
	for _, issue := range report.Issues {
		if issue.Message == "" || issue.Recommendation == "" || issue.Severity != "warning" {
			t.Errorf("incomplete issue: %+v", issue)
		}
	}
}

func TestEventQualityRules(t *testing.T) {
	tests := []struct {
		name, code string
		change     func(*model.AdminEvent)
	}{
		{"short unicode title", "title.length", func(e *model.AdminEvent) { e.Title = "ÄÖÜß🎵" }},
		{"long title", "title.length", func(e *model.AdminEvent) { e.Title = strings.Repeat("ä", 121) }},
		{"short description", "description.too_short", func(e *model.AdminEvent) { e.Description = qualityPtr("Ein Konzert.") }},
		{"single long sentence", "description.single_sentence", func(e *model.AdminEvent) { e.Description = qualityPtr(strings.Repeat("Musik und Kunst ", 50)) }},
		{"long description", "description.too_long", func(e *model.AdminEvent) { e.Description = qualityPtr(strings.Repeat("Musik. ", 1500)) }},
		{"many paragraphs", "description.too_many_paragraphs", func(e *model.AdminEvent) { e.Description = qualityPtr(strings.Repeat("<p>Musik und Kultur.</p>", 21)) }},
		{"many headings", "description.too_many_headings", func(e *model.AdminEvent) {
			e.Description = qualityPtr(strings.Repeat("<h2>Programm</h2><p>Musik.</p>", 9))
		}},
		{"short summary", "summary.length", func(e *model.AdminEvent) { e.Summary = qualityPtr("Musik.") }},
		{"long summary", "summary.length", func(e *model.AdminEvent) { e.Summary = qualityPtr(strings.Repeat("ä", 501)) }},
		{"summary paragraphs", "summary.too_many_paragraphs", func(e *model.AdminEvent) { e.Summary = qualityPtr("<p>Musik.</p><p>Kultur.</p>") }},
		{"summary breaks", "summary.too_many_paragraphs", func(e *model.AdminEvent) { e.Summary = qualityPtr("Musik<br>Kultur") }},
		{"summary emphasis ratio", "summary.too_much_emphasis", func(e *model.AdminEvent) { e.Summary = qualityPtr("<strong>" + *e.Summary + "</strong>") }},
		{"summary emphasis count", "summary.too_much_emphasis", func(e *model.AdminEvent) { e.Summary = qualityPtr(strings.Repeat("<b>A</b> ", 4) + *e.Summary) }},
		{"summary list", "summary.formatting", func(e *model.AdminEvent) { e.Summary = qualityPtr("<ul><li>Musik</li></ul>") }},
		{"summary style", "summary.formatting", func(e *model.AdminEvent) { e.Summary = qualityPtr(`<span style="color:red">Musik</span>`) }},
		{"small image", "image.low_resolution", func(e *model.AdminEvent) { e.Images[0].Height = qualityPtr(int32(629)) }},
		{"unknown resolution", "image.resolution_unknown", func(e *model.AdminEvent) { e.Images[0].Width = nil }},
		{"zero resolution", "image.resolution_unknown", func(e *model.AdminEvent) { e.Images[0].Width = qualityPtr(int32(0)) }},
		{"missing start date", "date.start_date_missing", func(e *model.AdminEvent) { e.EventDates[0].StartDate = nil }},
		{"missing start time", "date.start_time_missing", func(e *model.AdminEvent) { e.EventDates[0].StartTime = qualityPtr(" ") }},
		{"missing venue", "date.location_missing", func(e *model.AdminEvent) { e.VenueUuid = nil }},
		{"invalid date", "date.start_date_invalid", func(e *model.AdminEvent) { e.EventDates[0].StartDate = qualityPtr("2026-02-30") }},
		{"invalid time", "date.start_time_invalid", func(e *model.AdminEvent) { e.EventDates[0].StartTime = qualityPtr("25:00") }},
		{"earlier end date", "date.end_before_start", func(e *model.AdminEvent) { e.EventDates[0].EndDate = qualityPtr("2026-10-09") }},
		{"earlier end time", "date.end_before_start", func(e *model.AdminEvent) { e.EventDates[0].EndTime = qualityPtr("19:00") }},
		{"equal end time", "date.end_before_start", func(e *model.AdminEvent) { e.EventDates[0].EndTime = qualityPtr("20:00") }},
		{"nonpositive duration", "date.duration_invalid", func(e *model.AdminEvent) { e.EventDates[0].Duration = qualityPtr(int64(0)) }},
		{"missing end", "date.end_missing", func(e *model.AdminEvent) { e.EventDates[0].EndTime = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := completeQualityEvent()
			tt.change(&event)
			report := EvaluateEventQuality(event)
			issue := issueWithCode(report, tt.code)
			if issue == nil {
				t.Fatalf("missing %s: %+v", tt.code, report.Issues)
			}
			if strings.HasPrefix(tt.code, "date.") && issue.EventDateUuid != "date-1" {
				t.Errorf("date not identified: %+v", issue)
			}
			if strings.HasPrefix(tt.code, "image.") && issue.ImageUuid != "image-1" {
				t.Errorf("image not identified: %+v", issue)
			}
		})
	}
}

func TestEventQualityValidVariants(t *testing.T) {
	tests := []struct {
		name   string
		change func(*model.AdminEvent)
	}{
		{"title minimum", func(e *model.AdminEvent) { e.Title = strings.Repeat("ä", 10) }},
		{"title maximum", func(e *model.AdminEvent) { e.Title = strings.Repeat("ä", 120) }},
		{"summary minimum", func(e *model.AdminEvent) { e.Summary = qualityPtr(strings.Repeat("ä", 80)) }},
		{"summary maximum", func(e *model.AdminEvent) { e.Summary = qualityPtr(strings.Repeat("ä", 500)) }},
		{"portrait image", func(e *model.AdminEvent) {
			e.Images[0].Width, e.Images[0].Height = e.Images[0].Height, e.Images[0].Width
		}},
		{"date venue", func(e *model.AdminEvent) { e.VenueUuid = nil; e.EventDates[0].VenueUuid = qualityPtr("venue-2") }},
		{"online", func(e *model.AdminEvent) { e.VenueUuid = nil; e.OnlineLink = qualityPtr("https://example.org/live") }},
		{"all day", func(e *model.AdminEvent) {
			e.EventDates[0].AllDay = qualityPtr(true)
			e.EventDates[0].StartTime = nil
			e.EventDates[0].EndTime = nil
		}},
		{"overnight", func(e *model.AdminEvent) {
			e.EventDates[0].EndDate = qualityPtr("2026-10-11")
			e.EventDates[0].EndTime = qualityPtr("01:00")
		}},
		{"duration instead of end", func(e *model.AdminEvent) {
			e.EventDates[0].EndTime = nil
			e.EventDates[0].Duration = qualityPtr(int64(90))
		}},
		{"sparse emphasis", func(e *model.AdminEvent) { e.Summary = qualityPtr("<p><b>Musik</b> " + *e.Summary + "</p>") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := completeQualityEvent()
			tt.change(&event)
			report := EvaluateEventQuality(event)
			if report.Status != "ok" || report.Completeness.Score != 100 {
				t.Fatalf("unexpected issues: %+v", report)
			}
		})
	}
}

func TestEventQualityAllDatesAndImages(t *testing.T) {
	event := completeQualityEvent()
	event.EventDates = append(event.EventDates, model.AdminEventDate{Uuid: "date-2", StartDate: qualityPtr("2026-10-12")})
	event.Images = append(event.Images, model.Image{Uuid: "image-2", Width: qualityPtr(int32(320)), Height: qualityPtr(int32(240))})
	report := EvaluateEventQuality(event)
	if report.Completeness.Total != 12 || report.Completeness.Present != 11 || report.Completeness.Score != 92 {
		t.Fatalf("unexpected completeness: %+v", report.Completeness)
	}
	if issue := issueWithCode(report, "date.start_time_missing"); issue == nil || issue.EventDateUuid != "date-2" {
		t.Fatalf("second date not evaluated: %+v", report.Issues)
	}
	if issue := issueWithCode(report, "image.low_resolution"); issue == nil || issue.ImageUuid != "image-2" {
		t.Fatalf("second image not evaluated: %+v", report.Issues)
	}
}

func TestEventQualityPresenceIndependentOfQuality(t *testing.T) {
	event := completeQualityEvent()
	event.Images[0].Width = qualityPtr(int32(100))
	event.Title = "X"
	report := EvaluateEventQuality(event)
	if report.Completeness.Score != 100 || report.Status != "warnings" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestAnalyzeEventText(t *testing.T) {
	tests := []struct {
		name, input                      string
		characters, paragraphs, headings int
	}{
		{"unicode and entities", "<p>Ä &amp; Ö 🎵</p>", 7, 1, 0},
		{"empty HTML", "<p><br></p><p>&nbsp;</p>", 0, 0, 0},
		{"hidden content", "<script>fake content</script><style>fake</style><template>fake</template><p>Musik</p>", 5, 1, 0},
		{"nested blocks", "<div><p>Musik</p><p>Kultur</p></div>", 12, 2, 0},
		{"plain paragraphs", "Musik\r\n \r\nKultur", 12, 2, 0},
		{"line wrapping", "Musik\nKultur", 12, 1, 0},
		{"headings", "<h2>Musik</h2><p>Kultur</p>", 12, 2, 1},
		{"malformed HTML", "<p>Musik<p>Kultur", 12, 2, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := analyzeEventText(tt.input)
			if m.Characters != tt.characters || m.Paragraphs != tt.paragraphs || m.Headings != tt.headings {
				t.Fatalf("unexpected metrics: %+v", m)
			}
		})
	}
	m := analyzeEventText("<b><i>Musik</i></b> und Kultur. Noch ein Satz!")
	if m.EmphasisRatio > 1 || m.EmphasisCount != 2 || m.Sentences != 2 || m.OtherFormatting != 0 {
		t.Fatalf("unexpected nested formatting metrics: %+v", m)
	}
}

func TestEventQualityMarkdown(t *testing.T) {
	event := completeQualityEvent()
	event.Description = qualityPtr(strings.Repeat("## Programm\n\nMusik und Kunst.\n\n", 9))
	event.Summary = qualityPtr("**Ein Konzert im Kulturhaus**\n\n- _Musik_\n- [Programm](https://example.org)\n\n~~Weitere Angaben~~")
	report := EvaluateEventQuality(event)
	for _, code := range []string{"description.too_many_headings", "summary.too_many_paragraphs", "summary.too_much_emphasis", "summary.formatting"} {
		if issueWithCode(report, code) == nil {
			t.Errorf("missing Markdown issue %s: %+v", code, report.Issues)
		}
	}
	metrics := analyzeEventText("**Musik** und _Kunst_ mit [Programm](https://example.org).")
	if metrics.Characters != 29 || metrics.EmphasisCount != 2 || metrics.OtherFormatting != 1 {
		t.Fatalf("unexpected Markdown metrics: %+v", metrics)
	}
	plain := analyzeEventText(`Ein \*Stern\* und a_b_c.`)
	if plain.EmphasisCount != 0 || plain.Characters != 22 {
		t.Fatalf("escaped markers counted as emphasis: %+v", plain)
	}
}
