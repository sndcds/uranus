package service

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/sndcds/uranus/model"
)

func renderPtr[T any](v T) *T { return &v }
func socialEvent() model.ContentItem {
	return model.ContentItem{
		SourceType: "event", SourceUuid: "event-1", Language: "de", Title: "Jazzabend",
		Summary: renderPtr("Musik **live**."), Description: renderPtr("Do not use this description"),
		Dates:    []model.ContentDate{{Uuid: "date-1", StartDate: "2026-10-01", StartTime: renderPtr("19:30")}},
		Location: &model.ContentLocation{Name: "Kulturhaus", City: renderPtr("Flensburg")},
		Images:   []model.Image{{Identifier: "main", Url: "https://api.example.test/api/image/image-1", Width: renderPtr(int32(1600)), Height: renderPtr(int32(900))}},
	}
}
func TestSocialRenderPlatforms(t *testing.T) {
	for _, platform := range []string{"facebook", "instagram", "mastodon", "bluesky"} {
		t.Run(platform, func(t *testing.T) {
			renderer, err := NewSocialRenderer(platform, "https://events.example.test")
			if err != nil {
				t.Fatal(err)
			}
			item := socialEvent()
			before, _ := json.Marshal(item)
			got, err := renderer.Render(item)
			if err != nil {
				t.Fatal(err)
			}
			for _, text := range []string{"Jazzabend", "01.10.2026", "19:30", "Kulturhaus", "Flensburg", "Musik live.", "#Kulturbytes", "https://events.example.test/de/veranstaltung/event-1/202610011930"} {
				if !strings.Contains(got.Text, text) {
					t.Errorf("missing %q in %q", text, got.Text)
				}
			}
			if got.Platform != platform || got.ImageURL == "" || got.URL == "" || strings.Contains(got.Text, "**") || strings.Contains(got.Text, "Do not use") {
				t.Fatalf("unexpected preview: %+v", got)
			}
			second, err := renderer.Render(item)
			after, _ := json.Marshal(item)
			if err != nil || got != second || string(before) != string(after) {
				t.Fatal("rendering is not deterministic or modified content")
			}
		})
	}
}
func TestSocialRenderLocalization(t *testing.T) {
	for _, platform := range []string{"facebook", "instagram", "mastodon"} {
		for _, lang := range []string{"de", "da", "en"} {
			t.Run(platform+"/"+lang, func(t *testing.T) {
				renderer, _ := NewSocialRenderer(platform, "https://events.example.test/root/")
				item := socialEvent()
				item.Language = lang
				item.Title = map[string]string{"de": "Jazzabend", "da": "Jazzaften", "en": "Jazz evening"}[lang]
				item.Dates[0].EndDate = renderPtr("2026-10-02")
				item.TicketLink = renderPtr("https://tickets.test")
				got, err := renderer.Render(item)
				if err != nil {
					t.Fatal(err)
				}
				for _, expected := range []string{item.Title, map[string]string{"de": "bis", "da": "til", "en": "until"}[lang] + " 02.10.2026", map[string]string{"de": "Tickets", "da": "Billetter", "en": "Tickets"}[lang]} {
					if !strings.Contains(got.Text, expected) {
						t.Errorf("missing %q: %s", expected, got.Text)
					}
				}
				route := map[string]string{"de": "veranstaltung", "da": "begivenhed", "en": "event"}[lang]
				if !strings.Contains(got.URL, "/root/"+lang+"/"+route+"/") {
					t.Fatal(got.URL)
				}
				if lang != "de" && strings.Contains(got.Text, "Uhr") {
					t.Fatal(got.Text)
				}
			})
		}
	}
}
func TestSocialRenderLimits(t *testing.T) {
	for _, tc := range []struct {
		platform string
		limit    int
	}{{"facebook", 63206}, {"instagram", 2200}, {"mastodon", 500}, {"bluesky", 300}} {
		t.Run(tc.platform, func(t *testing.T) {
			renderer, _ := NewSocialRenderer(tc.platform, "")
			item := socialEvent()
			item.Summary = renderPtr(strings.Repeat("ü", tc.limit+1))
			if _, err := renderer.Render(item); err == nil || !strings.Contains(err.Error(), "limit") {
				t.Fatalf("expected limit error, got %v", err)
			}
			if err := validateSocialText(tc.platform, strings.Repeat("ü", tc.limit)); err != nil {
				t.Fatal(err)
			}
			if err := validateSocialText(tc.platform, strings.Repeat("ü", tc.limit+1)); err == nil {
				t.Fatal("accepted one character too many")
			}
		})
	}
	// Grapheme clusters, including combining marks and ZWJ emoji, are not runes.
	for _, text := range []string{strings.Repeat("e\u0301", 300), strings.Repeat("👩‍💻", 200)} {
		if err := validateSocialText("bluesky", text); err != nil {
			t.Fatal(err)
		}
	}
	if err := validateSocialText("bluesky", strings.Repeat("👩‍💻", 300)); err == nil || !strings.Contains(err.Error(), "bytes") {
		t.Fatal("Bluesky byte limit not enforced")
	}
}
func TestSocialRenderMastodonURLs(t *testing.T) {
	for _, link := range []string{"https://a.test", "https://example.test/" + strings.Repeat("a", 700)} {
		if err := validateSocialText("mastodon", strings.Repeat("ü", 476)+" "+link); err != nil {
			t.Fatal(err)
		}
		if err := validateSocialText("mastodon", strings.Repeat("ü", 477)+" "+link); err == nil {
			t.Fatal("URL must count as 23 characters")
		}
	}
	renderer, _ := NewSocialRenderer("mastodon", "")
	item := socialEvent()
	item.Url = renderPtr("https://example.test/" + strings.Repeat("a", 700))
	got, err := renderer.Render(item)
	if err != nil || !strings.Contains(got.Text, *item.Url) || got.URL != *item.Url {
		t.Fatalf("URL shortened or rejected: %v", err)
	}
}
func TestSocialRenderImages(t *testing.T) {
	item := socialEvent()
	item.Images = append([]model.Image{{Identifier: "gallery_image_1", Url: "https://wrong.test/image"}}, item.Images...)
	item.Images[1].Url += "?width=50&height=50&fit=contain&ratio=1:1&type=png&focus=0.25,0.75"
	for _, tc := range []struct{ platform, ratio, kind, edge, size string }{
		{"instagram", "4:5", "jpg", "width", "1080"}, {"facebook", "1200:630", "webp", "width", "1920"},
		{"mastodon", "", "jpg", "width", "1920"}, {"bluesky", "4:3", "jpg", "width", "1920"},
	} {
		renderer, _ := NewSocialRenderer(tc.platform, "")
		got, err := renderer.Render(item)
		if err != nil {
			t.Fatal(err)
		}
		u, _ := url.Parse(got.ImageURL)
		q := u.Query()
		if u.Host != "api.example.test" || q.Get("ratio") != tc.ratio || q.Get("type") != tc.kind || q.Get(tc.edge) != tc.size || q.Get("height") != "" || q.Get("fit") != "" || q.Get("focus") != "0.25,0.75" {
			t.Fatal(got.ImageURL)
		}
	}
	item.Images = item.Images[1:]
	item.Images[0].Width = renderPtr(int32(900))
	item.Images[0].Height = renderPtr(int32(1600))
	renderer, _ := NewSocialRenderer("mastodon", "")
	got, err := renderer.Render(item)
	if err != nil || !strings.Contains(got.ImageURL, "height=1920") || strings.Contains(got.ImageURL, "width=") {
		t.Fatal(got, err)
	}
	item.Images[0].Width = nil
	got, err = renderer.Render(item)
	if err != nil || strings.Contains(got.ImageURL, "height=") || strings.Contains(got.ImageURL, "width=") {
		t.Fatal(got, err)
	}
}
func TestSocialRenderMissingAndInvalidContent(t *testing.T) {
	if _, err := NewSocialRenderer("unknown", ""); err == nil {
		t.Fatal("accepted unknown platform")
	}
	for _, tc := range []struct {
		name   string
		change func(*model.ContentItem)
		reason string
	}{
		{"missing", func(c *model.ContentItem) { *c = model.ContentItem{} }, "content"},
		{"image", func(c *model.ContentItem) { c.Images = nil }, "image is required"},
		{"image URL", func(c *model.ContentItem) { c.Images[0].Url = "javascript:alert(1)" }, "image URL"},
		{"date", func(c *model.ContentItem) { c.Dates[0].StartDate = "2026-02-30" }, "start_date"},
		{"time", func(c *model.ContentItem) { c.Dates[0].StartTime = renderPtr("25:00") }, "start_time"},
		{"URL", func(c *model.ContentItem) { c.Url = renderPtr("bad") }, "content URL"},
		{"source", func(c *model.ContentItem) { c.SourceType = "bad" }, "source_type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			renderer, _ := NewSocialRenderer("instagram", "")
			item := socialEvent()
			tc.change(&item)
			if _, err := renderer.Render(item); err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("got %v", err)
			}
		})
	}
	for _, platform := range []string{"facebook", "mastodon", "bluesky"} {
		renderer, _ := NewSocialRenderer(platform, "")
		item := socialEvent()
		item.Images = nil
		if got, err := renderer.Render(item); err != nil || got.ImageURL != "" {
			t.Fatal(got, err)
		}
	}
}
func TestSocialRenderAllDatesAndSources(t *testing.T) {
	renderer, _ := NewSocialRenderer("facebook", "https://events.example.test")
	item := socialEvent()
	item.Dates[0].StartTime = nil
	item.Dates[0].AllDay = renderPtr(true)
	item.Dates = append(item.Dates, model.ContentDate{Uuid: "date-2", StartDate: "2026-10-03", Location: &model.ContentLocation{Name: "Other venue"}})
	got, err := renderer.Render(item)
	if err != nil || !strings.HasSuffix(got.URL, "/date-1") || !strings.Contains(got.Text, "03.10.2026\nOther venue") || strings.Contains(got.Text, "00:00") {
		t.Fatal(got, err)
	}
	for _, kind := range []string{"venue", "organization"} {
		item.SourceType = kind
		item.Dates = nil
		item.Url = renderPtr("https://source.example.test")
		got, err = renderer.Render(item)
		if err != nil || kind == "venue" && !strings.Contains(got.URL, "/de/ort/event-1") || kind == "organization" && got.URL != *item.Url {
			t.Fatal(got, err)
		}
	}
}
func TestSocialRenderAccountValidation(t *testing.T) {
	valid := model.SocialAccount{Uuid: "account", OrgUuid: "org", Platform: "mastodon", RemoteAccountID: "123", Enabled: true, BaseURL: renderPtr("https://social.example.test")}
	if err := ValidateSocialRenderAccount(valid, "org"); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*model.SocialAccount){
		func(a *model.SocialAccount) { a.Uuid = "" }, func(a *model.SocialAccount) { a.OrgUuid = "foreign" },
		func(a *model.SocialAccount) { a.Enabled = false }, func(a *model.SocialAccount) { a.RemoteAccountID = " " },
		func(a *model.SocialAccount) { a.BaseURL = nil }, func(a *model.SocialAccount) { a.BaseURL = renderPtr("https://user:secret@social.example.test") },
		func(a *model.SocialAccount) { a.BaseURL = renderPtr("https://social.example.test/path") },
	} {
		a := valid
		change(&a)
		if err := ValidateSocialRenderAccount(a, "org"); err == nil {
			t.Fatal("accepted invalid account")
		}
	}
}
func TestSocialPlainText(t *testing.T) {
	got := socialPlainText("## Musik\n\n**Jazz** &amp; [Tickets](https://tickets.test)\n\n<div>Live<br>heute</div><script>secret</script>")
	for _, expected := range []string{"Musik", "Jazz & Tickets (https://tickets.test)", "Live\nheute"} {
		if !strings.Contains(got, expected) {
			t.Errorf("missing %q in %q", expected, got)
		}
	}
	for _, forbidden := range []string{"**", "<div>", "secret", "##"} {
		if strings.Contains(got, forbidden) {
			t.Fatal(got)
		}
	}
}

func TestSocialRenderSummaryFallbackAndConfiguration(t *testing.T) {
	item := socialEvent()
	item.Summary = renderPtr(" \n ")
	item.Description = renderPtr("Full **description**")
	renderer, _ := NewSocialRenderer("facebook", "")
	got, err := renderer.Render(item)
	if err != nil || !strings.Contains(got.Text, "Full description") {
		t.Fatal(got, err)
	}
	renderer, _ = NewSocialRenderer("facebook", "not a URL")
	if _, err := renderer.Render(item); err == nil || !strings.Contains(err.Error(), "frontend-client") {
		t.Fatal(err)
	}
	for _, link := range []string{"https://example.test.", "HTTPS://example.test!", "https://example.test)"} {
		if err := validateSocialText("mastodon", strings.Repeat("a", 476)+" "+link); err == nil {
			t.Fatal("URL punctuation was not counted")
		}
	}
}
