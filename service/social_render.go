package service

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
	"github.com/sndcds/uranus/model"
)

// SocialRenderer only transforms canonical content. It has no database,
// credentials, HTTP client, clock or publishing dependency.
type SocialRenderer interface {
	Render(model.ContentItem) (model.RenderedPost, error)
}

type socialRenderer struct{ platform, frontend string }

func NewSocialRenderer(platform, frontend string) (SocialRenderer, error) {
	switch platform {
	case "facebook", "instagram", "mastodon", "bluesky":
		return socialRenderer{platform, frontend}, nil
	default:
		return nil, fmt.Errorf("unsupported platform")
	}
}

// ValidateSocialRenderAccount checks metadata only; preview never needs tokens.
func ValidateSocialRenderAccount(account model.SocialAccount, orgUUID string) error {
	if account.Uuid == "" {
		return fmt.Errorf("social account not found")
	}
	if account.OrgUuid != orgUUID {
		return fmt.Errorf("social account belongs to another organization")
	}
	if !account.Enabled {
		return fmt.Errorf("social account is disabled")
	}
	if strings.TrimSpace(account.RemoteAccountID) == "" {
		return fmt.Errorf("remote_account_id is required")
	}
	if account.Platform == "mastodon" || account.BaseURL != nil {
		u, err := socialURL(value(account.BaseURL))
		if err != nil || u.Path != "" && u.Path != "/" || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("base_url must be an HTTP(S) origin without credentials, path, query or fragment")
		}
	}
	return nil
}

func socialURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("invalid HTTP(S) URL")
	}
	return u, nil
}

type socialLabels struct{ until, timeSuffix, info, tickets string }

func labels(lang string) socialLabels {
	switch lang {
	case "da":
		return socialLabels{"til", "", "Flere oplysninger", "Billetter"}
	case "en":
		return socialLabels{"until", "", "More information", "Tickets"}
	default:
		return socialLabels{"bis", " Uhr", "Mehr Informationen", "Tickets"}
	}
}

func (r socialRenderer) Render(item model.ContentItem) (model.RenderedPost, error) {
	result := model.RenderedPost{Platform: r.platform}
	if strings.TrimSpace(item.Title) == "" || item.SourceUuid == "" {
		return result, fmt.Errorf("content title and source_uuid are required")
	}
	l := labels(item.Language)
	compact := r.platform != "facebook"
	title := socialPlainText(item.Title)
	if title == "" {
		return result, fmt.Errorf("content title contains no visible text")
	}
	if r.platform != "bluesky" {
		title = "📅 " + title
	}
	lines := []string{title}
	add := func(s string) {
		if s != "" {
			lines = append(lines, s)
		}
	}
	add(socialPlainText(value(item.Subtitle)))
	for _, date := range item.Dates {
		start, err := time.Parse("2006-01-02", date.StartDate)
		if err != nil {
			return result, fmt.Errorf("invalid start_date for date %s", date.Uuid)
		}
		dateText := start.Format("02.01.2006")
		if value(date.StartTime) != "" && (date.AllDay == nil || !*date.AllDay) {
			if _, err := time.Parse("15:04", *date.StartTime); err != nil {
				return result, fmt.Errorf("invalid start_time for date %s", date.Uuid)
			}
			dateText += " · " + *date.StartTime + l.timeSuffix
		}
		if value(date.EndDate) != "" && *date.EndDate != date.StartDate {
			end, err := time.Parse("2006-01-02", *date.EndDate)
			if err != nil {
				return result, fmt.Errorf("invalid end_date for date %s", date.Uuid)
			}
			dateText += " " + l.until + " " + end.Format("02.01.2006")
		}
		if r.platform != "bluesky" {
			dateText = "🗓 " + dateText
		}
		add(dateText)
		location := date.Location
		if location == nil {
			location = item.Location
		}
		add(renderSocialLocation(location, compact))
		add(socialPlainText(value(date.SpaceName)))
		if value(date.TicketLink) != "" && value(date.TicketLink) != value(item.TicketLink) {
			if _, err := socialURL(*date.TicketLink); err != nil {
				return result, fmt.Errorf("invalid ticket_link for date %s", date.Uuid)
			}
			add("🎟 " + l.tickets + ": " + *date.TicketLink)
		}
	}
	if len(item.Dates) == 0 {
		add(renderSocialLocation(item.Location, compact))
	}
	body := value(item.Summary)
	if body == "" {
		body = value(item.Description)
	}
	if body = socialPlainText(body); body != "" {
		add("\n" + body)
	}
	if value(item.TicketLink) != "" {
		if _, err := socialURL(*item.TicketLink); err != nil {
			return result, fmt.Errorf("invalid ticket_link")
		}
		add("\n🎟 " + l.tickets + ": " + *item.TicketLink)
	}
	link, err := r.contentURL(item)
	if err != nil {
		return result, err
	}
	result.URL = link
	if link != "" {
		prefix := "\n👉 "
		if r.platform == "facebook" {
			prefix += l.info + ": "
		}
		if r.platform == "bluesky" {
			prefix = "\n"
		}
		add(prefix + link)
	}
	tags := "#Kulturbytes"
	if item.Location != nil {
		city := strings.Map(func(c rune) rune {
			if unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' {
				return c
			}
			return -1
		}, value(item.Location.City))
		if city != "" && !strings.EqualFold(city, "Kulturbytes") {
			tags += " #" + city
		}
	}
	add(tags)
	result.Text = strings.Join(lines, "\n")
	if err := validateSocialText(r.platform, result.Text); err != nil {
		return result, err
	}
	result.ImageURL, err = r.imageURL(item)
	return result, err
}

func renderSocialLocation(loc *model.ContentLocation, compact bool) string {
	if loc == nil {
		return ""
	}
	parts := []string{socialPlainText(loc.Name)}
	if !compact {
		if street := strings.TrimSpace(value(loc.Street) + " " + value(loc.HouseNumber)); street != "" {
			parts = append(parts, socialPlainText(street))
		}
	}
	if city := socialPlainText(value(loc.City)); city != "" && !strings.EqualFold(city, parts[0]) {
		parts = append(parts, city)
	}
	return strings.Trim(strings.Join(parts, ", "), ", ")
}

// Client routes match kulturbytes-client's defineI18nRoute declarations.
// Use the existing date slug when possible. Without a time, the client's
// supported date UUID avoids inventing a midnight occurrence.
func (r socialRenderer) contentURL(item model.ContentItem) (string, error) {
	route := ""
	switch item.SourceType {
	case "event":
		if len(item.Dates) > 0 {
			route = map[string]string{"de": "veranstaltung", "da": "begivenhed", "en": "event"}[item.Language]
		}
	case "venue":
		route = map[string]string{"de": "ort", "da": "sted", "en": "venue"}[item.Language]
	case "organization": // The client has no public organization detail route.
	default:
		return "", fmt.Errorf("invalid content source_type")
	}
	if r.frontend != "" && route != "" {
		base, err := socialURL(r.frontend)
		if err != nil || base.RawQuery != "" || base.Fragment != "" {
			return "", fmt.Errorf("invalid frontend-client configuration")
		}
		link := strings.TrimRight(base.String(), "/") + "/" + item.Language + "/" + route + "/" + url.PathEscape(item.SourceUuid)
		if item.SourceType == "event" {
			date := item.Dates[0]
			identifier := date.Uuid
			if value(date.StartTime) != "" {
				identifier = strings.ReplaceAll(date.StartDate, "-", "") + strings.ReplaceAll(*date.StartTime, ":", "")
			}
			if identifier == "" {
				return "", fmt.Errorf("event date identifier is required for public URL")
			}
			link += "/" + url.PathEscape(identifier)
		}
		return link, nil
	}
	if value(item.Url) != "" {
		if _, err := socialURL(*item.Url); err != nil {
			return "", fmt.Errorf("invalid content URL")
		}
	}
	return value(item.Url), nil
}

// Limits are offline rendering policies. Mastodon uses the standard 500/23
// defaults; instance-specific discovery belongs outside this pure renderer.
var socialLinks = regexp.MustCompile(`(?i)https?://[^\s<>]+`)

func validateSocialText(platform, text string) error {
	count, limit := utf8.RuneCountInString(text), 63206
	switch platform {
	case "instagram":
		limit = 2200
	case "mastodon":
		limit = 500
		count = uniseg.GraphemeClusterCount(socialLinks.ReplaceAllStringFunc(text, func(link string) string {
			// Sentence punctuation and unmatched closing brackets are outside a URL.
			urlText := strings.TrimRight(link, ".,!?;:")
			for _, pair := range [][2]string{{"(", ")"}, {"[", "]"}, {"{", "}"}} {
				for strings.HasSuffix(urlText, pair[1]) && strings.Count(urlText, pair[1]) > strings.Count(urlText, pair[0]) {
					urlText = strings.TrimSuffix(urlText, pair[1])
				}
			}
			return strings.Repeat("x", 23) + link[len(urlText):]
		}))
	case "bluesky":
		count, limit = uniseg.GraphemeClusterCount(text), 300
		if len(text) > 3000 {
			return fmt.Errorf("text exceeds 3000 UTF-8 bytes (got %d)", len(text))
		}
	}
	if count > limit {
		return fmt.Errorf("text exceeds limit of %d characters (got %d); content was not truncated", limit, count)
	}
	return nil
}

// Only canonical Pluto images are used; generating URLs performs no I/O.
func (r socialRenderer) imageURL(item model.ContentItem) (string, error) {
	preferred := map[string]string{"event": "main", "venue": "main_photo", "organization": "main_logo"}[item.SourceType]
	var selected *model.Image
	for i := range item.Images {
		if item.Images[i].Identifier == preferred {
			selected = &item.Images[i]
			break
		}
	}
	if selected == nil && len(item.Images) > 0 {
		selected = &item.Images[0]
	}
	if selected == nil {
		if r.platform == "instagram" {
			return "", fmt.Errorf("an image is required")
		}
		return "", nil
	}
	u, err := socialURL(selected.Url)
	if err != nil {
		return "", fmt.Errorf("invalid image URL")
	}
	q := u.Query()
	for _, key := range []string{"width", "height", "ratio", "fit", "type"} {
		q.Del(key)
	}
	switch r.platform {
	case "instagram":
		q.Set("ratio", "4:5")
		q.Set("width", "1080")
		q.Set("type", "jpg")
	case "facebook":
		q.Set("ratio", "1200:630")
		q.Set("width", "1920")
		q.Set("type", "webp")
	case "bluesky":
		q.Set("ratio", "4:3")
		q.Set("width", "1920")
		q.Set("type", "jpg")
	case "mastodon":
		q.Set("type", "jpg")
		if selected.Width != nil && selected.Height != nil && *selected.Width > 0 && *selected.Height > 0 {
			if *selected.Width >= *selected.Height {
				q.Set("width", "1920")
			} else {
				q.Set("height", "1920")
			}
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
