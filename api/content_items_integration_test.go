package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

const (
	contentEvent      = "01994126-6680-7000-8000-000000000010"
	contentVenue      = "01994126-6680-7000-8000-000000000011"
	contentOtherVenue = "01994126-6680-7000-8000-000000000012"
	contentSpace      = "01994126-6680-7000-8000-000000000013"
	contentDate       = "01994126-6680-7000-8000-000000000014"
	contentOtherDate  = "01994126-6680-7000-8000-000000000015"
	contentImage      = "01994126-6680-7000-8000-000000000016"
)

func contentDatabase(t *testing.T) (*ApiHandler, *gin.Engine, string) {
	t.Helper()
	h, r, token := socialPostDatabase(t)
	app.UranusInstance.Config.SupportedLanguages = []string{"de", "da", "en"}
	app.UranusInstance.Config.BaseApiUrl = "https://api.example.test"
	// The repository's table DDL references these enums, but has no enum DDL.
	dbExec(t, h, "CREATE TYPE uranus.event_release_status AS ENUM ('draft','review','released','cancelled','deferred','rescheduled','inherited')")
	dbExec(t, h, "CREATE TYPE uranus.uranus_ticket_flag AS ENUM ('advance_ticket','presale_fee_applies','on_site_ticket_sales','reduced_price_available')")
	dbExec(t, h, "CREATE TYPE uranus.uranus_price_type AS ENUM ('not_specified','regular_price','free','donation','tiered_prices')")
	for _, table := range []string{"venue", "space", "event", "event_date", "license", "license_i18n", "pluto_image", "pluto_image_link"} {
		ddl, err := os.ReadFile("../ddl/" + table + ".ddl")
		if err != nil {
			t.Fatal(err)
		}
		dbExec(t, h, strings.Split(string(ddl), "-- Indices")[0])
	}

	dbExec(t, h, `UPDATE uranus.organization SET name='Kulturverein', description='Original **Text**',
		content_iso_639_1='de', web_link='https://example.test/org', city='Flensburg',
		api_import_token='secret-access' WHERE uuid=$1`, socialOrg)
	dbExec(t, h, `INSERT INTO uranus.venue (uuid,org_uuid,name,description,summary,content_iso_639_1,
		web_link,ticket_link,street,house_number,postal_code,city,state,country,scope)
		VALUES ($1,$2,'Kulturhaus','<p>Original venue</p>','Venue summary','da',
		'https://example.test/venue','https://example.test/tickets','Straße','7','24937','Flensburg','SH','DEU','organization')`, contentVenue, socialOrg)
	// Related venues can legitimately belong to another organization.
	dbExec(t, h, `INSERT INTO uranus.venue (uuid,org_uuid,name,city,scope)
		VALUES ($1,$2,'Partner venue','Husum','shared')`, contentOtherVenue, socialOtherOrg)
	dbExec(t, h, "INSERT INTO uranus.space (uuid,venue_uuid,name) VALUES ($1,$2,'Main room')", contentSpace, contentVenue)
	dbExec(t, h, `INSERT INTO uranus.event (uuid,org_uuid,venue_uuid,space_uuid,content_iso_639_1,
		title,subtitle,description,summary,source_link,ticket_link,online_link)
		VALUES ($1,$2,$3,$4,'de','Konzert','Live','**Original event**','Event summary',
		'https://example.test/event','https://example.test/event-tickets','https://example.test/stream')`,
		contentEvent, socialOrg, contentVenue, contentSpace)
	dbExec(t, h, `INSERT INTO uranus.event_date (uuid,event_uuid,start_date,start_time,end_date,end_time,
		entry_time,duration,all_day) VALUES ($1,$2,'2027-01-02','19:30','2027-01-03','01:00','19:00',330,false)`, contentDate, contentEvent)
	dbExec(t, h, `INSERT INTO uranus.event_date (uuid,event_uuid,start_date,venue_uuid,all_day,ticket_link)
		VALUES ($1,$2,'2027-01-03',$3,true,'https://example.test/date-tickets')`, contentOtherDate, contentEvent, contentOtherVenue)
	dbExec(t, h, "INSERT INTO uranus.license (key) VALUES ('cc-by-4.0'),('all-rights-reserved')")
	for _, lang := range []string{"de", "da", "en"} {
		dbExec(t, h, `INSERT INTO uranus.license_i18n (key,iso_639_1,name,description)
			VALUES ('cc-by-4.0',$1,$2,$3),('all-rights-reserved',$1,$4,$5)`,
			lang, "CC "+lang, "CC description "+lang, "Reserved "+lang, "Reserved description "+lang)
	}
	dbExec(t, h, `INSERT INTO uranus.pluto_image (uuid,file_name,alt_text,description,width,height,
		creator_name,copyright,license,focus_x,focus_y,ai_label)
		VALUES ($1,'photo.jpg','Original alt','Original image description',1600,900,'Photographer',
		'Copyright holder','cc-by-4.0',0.25,0.75,'ai_modified')`, contentImage)
	for _, source := range []struct{ kind, id, identifier string }{
		{"event", contentEvent, "main"}, {"venue", contentVenue, "main_photo"}, {"organization", socialOrg, "main_logo"},
	} {
		dbExec(t, h, `INSERT INTO uranus.pluto_image_link (pluto_image_uuid,context,context_uuid,identifier)
			VALUES ($1,$2,$3,$4)`, contentImage, source.kind, source.id, source.identifier)
	}
	return h, r, token
}

func loadContent(t *testing.T, h *ApiHandler, sourceType, sourceUuid, lang string) *model.ContentItem {
	t.Helper()
	item, txErr := h.LoadContentItem(contentContext(authTestUser), socialOrg, sourceType, sourceUuid, lang)
	if txErr != nil {
		t.Fatalf("load content: %v", txErr)
	}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret-access", "secret-refresh", "api_import_token", "access_token", "refresh_token"} {
		if strings.Contains(string(data), secret) {
			t.Fatal("content exposed a secret")
		}
	}
	if item.SourceType != sourceType || item.SourceUuid != sourceUuid || item.OrgUuid != socialOrg || item.Dates == nil || item.Images == nil {
		t.Fatalf("invalid source identity or collections: %+v", item)
	}
	return item
}

func TestContentItemPostgresEvent(t *testing.T) {
	h, _, _ := contentDatabase(t)
	item := loadContent(t, h, "event", contentEvent, "en")
	if item.Title != "Konzert" || *item.Subtitle != "Live" || *item.Description != "**Original event**" ||
		*item.Summary != "Event summary" || *item.Url != "https://example.test/event" ||
		*item.OnlineLink != "https://example.test/stream" || *item.ContentLanguage != "de" ||
		item.Language != "en" || item.Location.Uuid != contentVenue || len(item.Dates) != 2 {
		t.Fatalf("unexpected event: %+v", item)
	}
	date := item.Dates[0]
	if date.Uuid != contentDate || date.StartDate != "2027-01-02" || *date.StartTime != "19:30" ||
		*date.EndDate != "2027-01-03" || *date.EndTime != "01:00" || *date.EntryTime != "19:00" ||
		*date.Duration != 330 || *date.AllDay || date.Location.Uuid != contentVenue ||
		*date.SpaceUuid != contentSpace || *date.SpaceName != "Main room" || date.TicketLink != nil {
		t.Fatalf("event/date inheritance failed: %+v", date)
	}
	date = item.Dates[1]
	if date.Uuid != contentOtherDate || date.StartTime != nil || date.EndDate != nil || date.EndTime != nil ||
		date.AllDay == nil || !*date.AllDay || date.Location.Uuid != contentOtherVenue || date.Location.Name != "Partner venue" ||
		date.SpaceUuid != nil || date.SpaceName != nil || *date.TicketLink != "https://example.test/date-tickets" {
		t.Fatalf("date override or all-day values failed: %+v", date)
	}
	// An explicit date venue with its own space follows the public detail query.
	dbExec(t, h, "UPDATE uranus.event_date SET venue_uuid=$1,space_uuid=$2 WHERE uuid=$3", contentVenue, contentSpace, contentOtherDate)
	date = loadContent(t, h, "event", contentEvent, "de").Dates[1]
	if date.Location.Uuid != contentVenue || *date.SpaceUuid != contentSpace {
		t.Fatal("explicit date space was not loaded")
	}
	// Load fresh source data without requiring a projection refresh.
	dbExec(t, h, "UPDATE uranus.event SET title='Changed' WHERE uuid=$1", contentEvent)
	if loadContent(t, h, "event", contentEvent, "de").Title != "Changed" {
		t.Fatal("source update was not reflected")
	}
}

func TestContentItemPostgresVenueAndOrganization(t *testing.T) {
	h, _, _ := contentDatabase(t)
	venue := loadContent(t, h, "venue", contentVenue, "de")
	if venue.Title != "Kulturhaus" || *venue.Description != "<p>Original venue</p>" || *venue.Summary != "Venue summary" ||
		*venue.Url != "https://example.test/venue" || *venue.TicketLink != "https://example.test/tickets" ||
		*venue.ContentLanguage != "da" || venue.Subtitle != nil || len(venue.Dates) != 0 {
		t.Fatalf("unexpected venue: %+v", venue)
	}
	location := venue.Location
	if location.Uuid != contentVenue || location.Name != venue.Title || *location.Street != "Straße" ||
		*location.HouseNumber != "7" || *location.PostalCode != "24937" || *location.City != "Flensburg" ||
		*location.State != "SH" || *location.Country != "DEU" || *location.Url != *venue.Url {
		t.Fatalf("unexpected location: %+v", location)
	}
	org := loadContent(t, h, "organization", socialOrg, "da")
	if org.Title != "Kulturverein" || *org.Description != "Original **Text**" || *org.Url != "https://example.test/org" ||
		org.Location.Uuid != socialOrg || *org.Location.City != "Flensburg" || org.Summary != nil || org.TicketLink != nil ||
		org.Subtitle != nil || len(org.Dates) != 0 {
		t.Fatalf("unexpected organization: %+v", org)
	}
}

func TestContentItemPostgresLocalizationAndImages(t *testing.T) {
	h, _, _ := contentDatabase(t)
	for _, source := range []struct{ kind, id, identifier string }{
		{"event", contentEvent, "main"}, {"venue", contentVenue, "main_photo"}, {"organization", socialOrg, "main_logo"},
	} {
		for _, lang := range []string{"de", "da", "en"} {
			item := loadContent(t, h, source.kind, source.id, lang)
			if item.Language != lang || len(item.Images) != 1 {
				t.Fatalf("incorrect locale or images: %+v", item)
			}
			image := item.Images[0]
			if image.Uuid != contentImage || image.Identifier != source.identifier ||
				image.Url != "https://api.example.test/api/image/"+contentImage || *image.Alt != "Original alt" ||
				*image.Width != 1600 || *image.Height != 900 || *image.Creator != "Photographer" ||
				*image.Copyright != "Copyright holder" || *image.Description != "Original image description" ||
				*image.License != "cc-by-4.0" || *image.LicenseName != "CC "+lang ||
				*image.LicenseDescription != "CC description "+lang || *image.AiLabel != "ai_modified" ||
				*image.FocusX != 0.25 || *image.FocusY != 0.75 {
				t.Fatalf("unexpected image metadata: %+v", image)
			}
		}
	}
	for _, lang := range []string{"", "unsupported"} {
		if item := loadContent(t, h, "event", contentEvent, lang); item.Language != "de" || *item.Images[0].LicenseName != "CC de" {
			t.Fatal("existing locale normalization was not used")
		}
	}
	// Use the same fallback license in the requested language as get-event.sql.
	dbExec(t, h, "UPDATE uranus.pluto_image SET license=NULL,ai_label=NULL WHERE uuid=$1", contentImage)
	image := loadContent(t, h, "event", contentEvent, "en").Images[0]
	if *image.License != "all-rights-reserved" || *image.LicenseName != "Reserved en" || *image.AiLabel != "none" {
		t.Fatal("license/AI-label fallback failed")
	}
	dbExec(t, h, "DELETE FROM uranus.license_i18n WHERE iso_639_1='da'")
	image = loadContent(t, h, "event", contentEvent, "da").Images[0]
	if image.License != nil || image.LicenseName != nil || image.LicenseDescription != nil {
		t.Fatal("missing localization was invented")
	}
}

func TestContentItemPostgresMissingData(t *testing.T) {
	h, _, _ := contentDatabase(t)
	dbExec(t, h, "DELETE FROM uranus.event_date")
	dbExec(t, h, "DELETE FROM uranus.pluto_image")
	dbExec(t, h, `UPDATE uranus.event SET venue_uuid=NULL,space_uuid=NULL,description=NULL,summary=NULL,
		subtitle=NULL,content_iso_639_1=NULL,source_link=NULL,ticket_link=NULL,online_link=NULL`)
	item := loadContent(t, h, "event", contentEvent, "de")
	if item.Title != "Konzert" || item.Location != nil || item.Description != nil || item.Summary != nil ||
		item.Subtitle != nil || item.ContentLanguage != nil || item.Url != nil || item.TicketLink != nil || item.OnlineLink != nil ||
		len(item.Dates) != 0 || len(item.Images) != 0 {
		t.Fatalf("missing event values were invented: %+v", item)
	}
	dbExec(t, h, "UPDATE uranus.venue SET description=NULL,summary=NULL,web_link=NULL,content_iso_639_1=NULL")
	dbExec(t, h, "UPDATE uranus.organization SET description=NULL,web_link=NULL,content_iso_639_1=NULL")
	for _, source := range []struct{ kind, id string }{{"venue", contentVenue}, {"organization", socialOrg}} {
		item := loadContent(t, h, source.kind, source.id, "de")
		if item.Description != nil || item.Summary != nil || item.Url != nil || item.ContentLanguage != nil || len(item.Images) != 0 {
			t.Fatalf("missing source values were invented: %+v", item)
		}
	}
}

func TestContentItemPostgresAuthorization(t *testing.T) {
	h, _, _ := contentDatabase(t)
	for _, sourceType := range []string{"event", "venue", "organization"} {
		item, txErr := h.LoadContentItem(contentContext(authTestUser), socialOrg, sourceType, socialOtherUser, "de")
		if item != nil || txErr == nil || txErr.Code != 404 {
			t.Fatalf("missing source: item=%+v err=%+v", item, txErr)
		}
	}
	for _, source := range []struct{ kind, id string }{{"event", contentEvent}, {"venue", contentVenue}, {"organization", socialOrg}} {
		// Even an administrator of both organizations cannot use a foreign source.
		item, txErr := h.LoadContentItem(contentContext(authTestUser), socialOtherOrg, source.kind, source.id, "de")
		if item != nil || txErr == nil || txErr.Code != 404 {
			t.Fatalf("foreign source: item=%+v err=%+v", item, txErr)
		}
	}
	assertDenied := func() {
		t.Helper()
		item, txErr := h.LoadContentItem(contentContext(socialOtherUser), socialOrg, "event", contentEvent, "de")
		if item != nil || txErr == nil || txErr.Code != 403 {
			t.Fatalf("unauthorized source: item=%+v err=%+v", item, txErr)
		}
	}
	assertDenied()
	dbExec(t, h, "INSERT INTO uranus.user_organization_link (user_uuid,org_uuid,permissions) VALUES ($1,$2,$3)", socialOtherUser, socialOrg, int64(app.UserPermEditOrg))
	dbExec(t, h, "INSERT INTO uranus.organization_member_link (user_uuid,org_uuid,has_joined) VALUES ($1,$2,false)", socialOtherUser, socialOrg)
	assertDenied()
	dbExec(t, h, "UPDATE uranus.organization_member_link SET has_joined=true WHERE user_uuid=$1", socialOtherUser)
	dbExec(t, h, "UPDATE uranus.user_organization_link SET permissions=0 WHERE user_uuid=$1", socialOtherUser)
	assertDenied()
	dbExec(t, h, "UPDATE uranus.user_organization_link SET permissions=$1 WHERE user_uuid=$2", int64(app.UserPermEditOrg), socialOtherUser)
	if _, txErr := h.LoadContentItem(contentContext(socialOtherUser), socialOrg, "event", contentEvent, "de"); txErr != nil {
		t.Fatal(txErr)
	}
}

func TestContentItemPostgresSocialPost(t *testing.T) {
	h, r, token := contentDatabase(t)
	account := createPostAccount(t, r, token, "facebook", socialOrg)
	body := strings.Replace(socialPostBody(socialOrg, "event", account), authTestUser, contentEvent, 1)
	w := socialRequest(r, "POST", socialPostsPath, token, body)
	assertSocialStatus(t, w, 201)
	post := socialPostData(t, w)
	dbExec(t, h, "UPDATE uranus.social_post_target SET status='scheduled',scheduled_at='2027-01-01T12:00:00Z' WHERE social_post_uuid=$1", post.Uuid)
	path := socialPostsPath + "/" + post.Uuid
	w = socialRequest(r, "GET", path, token, "")
	assertSocialStatus(t, w, 200)
	before := socialPostData(t, w)
	item, txErr := h.LoadSocialPostContentItem(contentContext(authTestUser), post.Uuid, "da")
	if txErr != nil || !reflect.DeepEqual(item, loadContent(t, h, "event", contentEvent, "da")) {
		t.Fatalf("post source resolution failed: item=%+v err=%+v", item, txErr)
	}
	w = socialRequest(r, "GET", path, token, "")
	assertSocialStatus(t, w, 200)
	if !reflect.DeepEqual(before, socialPostData(t, w)) {
		t.Fatal("loading content mutated the post or targets")
	}
	if item, txErr := h.LoadSocialPostContentItem(contentContext(socialOtherUser), post.Uuid, "de"); item != nil || txErr == nil || txErr.Code != 403 {
		t.Fatalf("unauthorized post: item=%+v err=%+v", item, txErr)
	}
	if item, txErr := h.LoadSocialPostContentItem(contentContext(authTestUser), socialOtherUser, "de"); item != nil || txErr == nil || txErr.Code != 404 {
		t.Fatalf("missing post: item=%+v err=%+v", item, txErr)
	}
	// Part 2 continues to accept valid UUID references without a source FK.
	for _, update := range []string{
		`{"source_type":"venue","source_uuid":"` + contentOtherVenue + `"}`,
		`{"source_type":"event","source_uuid":"` + socialOtherUser + `"}`,
	} {
		assertSocialStatus(t, socialRequest(r, "PUT", path, token, update), 200)
		if item, txErr := h.LoadSocialPostContentItem(contentContext(authTestUser), post.Uuid, "de"); item != nil || txErr == nil || txErr.Code != 404 {
			t.Fatalf("foreign/missing post source: item=%+v err=%+v", item, txErr)
		}
	}
}

func TestContentItemPostgresFailedRelations(t *testing.T) {
	h, _, _ := contentDatabase(t)
	captureSocialLogs(t)
	// A failed relation query must not return partially prepared content.
	for _, table := range []string{"event_date", "pluto_image_link"} {
		dbExec(t, h, "ALTER TABLE uranus."+table+" RENAME TO hidden_"+table)
		item, txErr := h.LoadContentItem(contentContext(authTestUser), socialOrg, "event", contentEvent, "de")
		if item != nil || txErr == nil || txErr.Code != http.StatusInternalServerError || txErr.Err != nil {
			t.Fatalf("partial content returned: item=%+v err=%+v", item, txErr)
		}
		dbExec(t, h, "ALTER TABLE uranus.hidden_"+table+" RENAME TO "+table)
	}
	// Cancelled requests return no content and do not retain raw DB errors.
	gc := contentContext(authTestUser)
	ctx, cancel := context.WithCancel(gc.Request.Context())
	cancel()
	gc.Request = gc.Request.WithContext(ctx)
	if item, txErr := h.LoadContentItem(gc, socialOrg, "event", contentEvent, "de"); item != nil || txErr == nil || txErr.Code != 500 || txErr.Err != nil {
		t.Fatalf("cancelled load: item=%+v err=%+v", item, txErr)
	}
}
