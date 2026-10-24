package api

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

func TestAdminEventListMetadataPostgres(t *testing.T) {
	h, router, token := contentDatabase(t)
	for _, table := range []string{"event_type_link", "event_link"} {
		ddl, err := os.ReadFile("../ddl/" + table + ".ddl")
		if err != nil {
			t.Fatal(err)
		}
		dbExec(t, h, strings.Split(string(ddl), "-- Indices")[0])
	}
	query, err := os.ReadFile("../sql/admin-get-org-events.sql")
	if err != nil {
		t.Fatal(err)
	}
	app.UranusInstance.SqlAdminGetOrgEvents = strings.NewReplacer("{{schema}}", h.DbSchema, "{{base_api_url}}", "https://api.example.test").Replace(string(query))
	router.GET("/api/admin/org/:orgUuid/events", app.JWTMiddleware, h.AdminGetOrgEvents)

	dbExec(t, h, `UPDATE uranus.event SET languages=ARRAY['de','da'], tags=ARRAY['Gospel','Live'],
        registration_deadline=CURRENT_DATE+7, min_price=0, max_price=20, currency='EUR',
        price_type='regular_price' WHERE uuid=$1`, contentEvent)
	dbExec(t, h, `UPDATE uranus.event_date SET start_date=CURRENT_DATE+1,end_date=CURRENT_DATE+1 WHERE uuid=$1`, contentDate)
	dbExec(t, h, `UPDATE uranus.event_date SET start_date=CURRENT_DATE+2,end_date=NULL,release_status='cancelled' WHERE uuid=$1`, contentOtherDate)
	dbExec(t, h, `INSERT INTO uranus.event_type_link(event_uuid,type_id,genre_id) VALUES ($1,1,1),($1,1,2)`, contentEvent)
	dbExec(t, h, `INSERT INTO uranus.event_link(event_uuid,url) VALUES ($1,'https://example.test/a'),($1,'https://example.test/b')`, contentEvent)
	const blank = "01994126-6680-7000-8000-000000000099"
	dbExec(t, h, `INSERT INTO uranus.event(uuid,org_uuid,title,description,summary) VALUES ($1,$2,'',E' \n\t',E' \n\t')`, blank, socialOrg)
	response := socialRequest(router, "GET", "/api/admin/org/"+socialOrg+"/events", token, "")
	assertSocialStatus(t, response, 200)
	var payload struct {
		Data []model.AdminListEvent `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data) != 3 {
		t.Fatalf("expected undated draft and two occurrences, got %d: %s", len(payload.Data), response.Body.String())
	}
	draft := payload.Data[0]
	if draft.Uuid != blank || draft.DateUuid != nil || draft.SeriesTotal != 0 || draft.ImageCount != 0 || draft.DescriptionLength != 0 || draft.HasRegistration || draft.HasTicket {
		t.Fatalf("absent metadata or undated ordering lost: %+v", draft)
	}
	first, second := payload.Data[1], payload.Data[2]
	if first.SeriesIndex != 1 || first.SeriesTotal != 2 || second.SeriesIndex != 2 || second.SeriesTotal != 2 {
		t.Fatalf("occurrence counts changed: %+v / %+v", first, second)
	}
	if first.ImageCount != 1 || first.LinkCount != 2 || first.TagCount != 2 || first.DescriptionLength != len("**Original event**") || first.SummaryLength != len("Event summary") {
		t.Fatalf("metadata counts multiplied by date/type joins: %+v", first)
	}
	if len(first.EventTypes) != 2 || len(first.Languages) != 2 || !first.HasRegistration || !first.HasTicket || first.MinPrice == nil || *first.MinPrice != 0 {
		t.Fatalf("event metadata missing: %+v", first)
	}
	if second.ReleaseStatus == nil || *second.ReleaseStatus != "cancelled" || !second.AllDay || second.ImageCount != first.ImageCount {
		t.Fatalf("date overrides must not change metadata: %+v", second)
	}
	// New zero/false fields must be serialized, unlike fields not supported by an old server.
	var raw struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"image_count", "description_length", "summary_length", "has_registration", "has_ticket", "series_total", "categories", "subtitle", "event_types"} {
		if _, ok := raw.Data[0][key]; !ok {
			t.Fatalf("zero/false field %s omitted", key)
		}
	}
}
