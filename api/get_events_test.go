package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
)

func TestGetEventsRequiresCompleteCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := &ApiHandler{}
	router.GET("/api/events", h.GetEvents)
	router.POST("/api/events/filter", h.GetEvents)
	for _, field := range []string{"last_event_start_at", "last_event_date_uuid"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			t.Run(method+"/"+field, func(t *testing.T) {
				var request *http.Request
				if method == http.MethodGet {
					request = httptest.NewRequest(method, "/api/events?"+field+"=incomplete", nil)
				} else {
					request = httptest.NewRequest(method, "/api/events/filter", strings.NewReader(`{"`+field+`":"incomplete"}`))
					request.Header.Set("Content-Type", "application/json")
				}
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "must be provided together") {
					t.Fatalf("incomplete cursor must return 400: %d %s", response.Code, response.Body.String())
				}
			})
		}
	}
}

// Only an explicitly supplied test database is used. Each test owns its schema.
func eventsDatabase(t *testing.T) (*ApiHandler, *gin.Engine) {
	t.Helper()
	dsn := os.Getenv("URANUS_EVENTS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set URANUS_EVENTS_TEST_DATABASE_URL to run PostgreSQL event pagination tests")
	}
	h := setupAuthTest(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h.DbPool = pool
	h.DbSchema = "events_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := pool.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS postgis"); err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, "CREATE SCHEMA "+h.DbSchema)
	t.Cleanup(func() { dbExec(t, h, "DROP SCHEMA "+h.DbSchema+" CASCADE") })
	fixture, err := os.ReadFile("testdata/get_events.sql")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, string(fixture))
	for path, destination := range map[string]*string{
		"get-events-projected.sql":              &app.UranusInstance.SqlGetEventsProjected,
		"get-events-geojson.sql":                &app.UranusInstance.SqlGetEventsGeoJSON,
		"resolve-event-date-uuid-from-slug.sql": &app.UranusInstance.SqlResolveEventDateUuidFromSlug,
	} {
		query, err := os.ReadFile("../sql/" + path)
		if err != nil {
			t.Fatal(err)
		}
		*destination = strings.ReplaceAll(string(query), "{{schema}}", h.DbSchema)
	}
	router := gin.New()
	router.GET("/api/events", h.GetEvents)
	router.POST("/api/events/filter", h.GetEvents)
	router.GET("/api/events/type-summary", h.GetEventTypeSummary)
	router.GET("/api/events/geojson", h.GetEventsGeoJSON)
	return h, router
}

func projectedDateUUID(id int) string {
	return fmt.Sprintf("01994126-6680-7000-8000-%012d", id)
}

func insertProjectedDate(t *testing.T, h *ApiHandler, id int, start, startTime, end string) {
	t.Helper()
	dateUUID := projectedDateUUID(id)
	eventUUID := projectedDateUUID(id + 100000)
	dbExec(t, h, `INSERT INTO uranus.event_projection (event_uuid,org_uuid,title)
		VALUES ($1,$2,$3)`, eventUUID, authTestUser, fmt.Sprintf("Event %d", id))
	dbExec(t, h, `INSERT INTO uranus.event_date_projection
		(event_date_uuid,event_uuid,start_date,start_time,end_date,all_day)
		VALUES ($1,$2,$3,NULLIF($4,'')::time,NULLIF($5,'')::date,$6)`,
		dateUUID, eventUUID, start, startTime, end, startTime == "")
}

func getEventsPage(t *testing.T, router *gin.Engine, method string, params url.Values) eventsResponse {
	t.Helper()
	var request *http.Request
	if method == http.MethodGet {
		request = httptest.NewRequest(method, "/api/events?"+params.Encode(), nil)
	} else {
		body := make(map[string]any)
		for key := range params {
			switch key {
			case "limit":
				body[key], _ = strconv.Atoi(params.Get(key))
			case "categories":
				body[key] = []int{1}
			default:
				body[key] = params.Get(key)
			}
		}
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		request = httptest.NewRequest(method, "/api/events/filter", strings.NewReader(string(encoded)))
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("event request failed: %d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), `"pagination_sort_at"`) {
		t.Fatal("internal sort value must not be exposed on events")
	}
	var payload struct {
		Data eventsResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Data
}

func TestGetEventsPostgresPagination(t *testing.T) {
	h, router := eventsDatabase(t)
	insertProjectedDate(t, h, 1, "2026-10-30", "09:00", "")
	insertProjectedDate(t, h, 2, "2026-10-25", "10:00", "2026-10-30") // Exactly five days: original sort date.
	insertProjectedDate(t, h, 25, "2026-10-30", "23:59", "")
	want := []string{projectedDateUUID(2), projectedDateUUID(1), projectedDateUUID(25)}
	for id := 100; id < 125; id++ {
		insertProjectedDate(t, h, id, "2026-10-01", "09:00", "2026-11-15")
		want = append(want, projectedDateUUID(id))
	}
	insertProjectedDate(t, h, 125, "2026-10-30", "09:00", "2026-11-05") // Starts on the reference date.
	insertProjectedDate(t, h, 126, "2026-10-01", "09:00", "2026-10-30") // Ends on the reference date.
	insertProjectedDate(t, h, 998, "2026-11-01", "09:00", "2026-11-20") // Future long event keeps its start.
	insertProjectedDate(t, h, 999, "2026-10-31", "19:00", "")
	insertProjectedDate(t, h, 1000, "2026-10-01", "09:00", "2026-10-20") // Already ended: excluded.
	want = append(want, projectedDateUUID(125), projectedDateUUID(126), projectedDateUUID(999), projectedDateUUID(998))
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, size := range []int{1, 7, 20} {
			t.Run(fmt.Sprintf("%s/limit=%d", method, size), func(t *testing.T) {
				params := url.Values{"start": {"2026-10-30"}, "end": {"2026-11-01"}, "limit": {strconv.Itoa(size)}, "categories": {"1"}}
				var got []string
				seen := make(map[string]bool)
				for page := 0; page <= len(want); page++ {
					data := getEventsPage(t, router, method, params)
					if data.Events == nil || data.PaginationReferenceDate != "2026-10-30" {
						t.Fatalf("events array or stable reference date lost: %+v", data)
					}
					if len(data.Events) == 0 {
						if data.LastEventStartAt != nil || data.LastEventDateUuid != nil {
							t.Fatal("empty page must terminate pagination with null cursors")
						}
						if !reflect.DeepEqual(got, want) {
							t.Fatalf("missing or misordered dates:\ngot  %v\nwant %v", got, want)
						}
						return
					}
					for _, event := range data.Events {
						if seen[event.DateUuid] {
							t.Fatalf("page %d repeats date %s", page+1, event.DateUuid)
						}
						seen[event.DateUuid] = true
						got = append(got, event.DateUuid)
					}
					last := data.Events[len(data.Events)-1]
					if data.LastEventStartAt == nil || data.LastEventDateUuid == nil || *data.LastEventDateUuid != last.DateUuid {
						t.Fatalf("cursor must identify the last returned date: %+v", data)
					}
					if _, err := time.Parse("2006-01-02T15:04", *data.LastEventStartAt); err != nil {
						t.Fatalf("cursor must have minute precision: %v", err)
					}
					if last.StartDate == "2026-10-01" && *data.LastEventStartAt != "2026-10-30T23:59" {
						t.Fatalf("long event must use the calculated cursor: %+v", data)
					}
					params.Set("start", data.PaginationReferenceDate)
					params.Set("last_event_start_at", *data.LastEventStartAt)
					params.Set("last_event_date_uuid", *data.LastEventDateUuid)
				}
				t.Fatal("pagination did not terminate")
			})
		}
	}
}

func TestGetEventsPostgresOriginalDatesAndSlugs(t *testing.T) {
	h, router := eventsDatabase(t)
	insertProjectedDate(t, h, 1, "2026-10-01", "09:00", "2026-11-15")
	insertProjectedDate(t, h, 2, "2026-10-01", "", "2026-11-15")
	data := getEventsPage(t, router, http.MethodGet, url.Values{"start": {"2026-10-30"}})
	if len(data.Events) != 2 {
		t.Fatalf("expected both long events, including one without a time: %+v", data)
	}
	first, second := data.Events[0], data.Events[1]
	if first.StartDate != "2026-10-01" || first.StartTime != "09:00" || first.DateSlug != "202610010900" || first.EndDate == nil || *first.EndDate != "2026-11-15" {
		t.Fatalf("pagination changed the original date: %+v", first)
	}
	resolved, err := h.ResolveEventDateUuidFromSlug(context.Background(), first.Uuid, first.DateSlug)
	if err != nil || resolved != first.DateUuid {
		t.Fatalf("detail slug does not resolve to the original date: %s %v", resolved, err)
	}
	if second.StartDate != "2026-10-01" || second.StartTime != "" || second.DateSlug != second.DateUuid || second.AllDay == nil || !*second.AllDay {
		t.Fatalf("date without time needs its original date and a UUID detail link: %+v", second)
	}
}

func TestGetEventsPostgresMinutePrecision(t *testing.T) {
	h, router := eventsDatabase(t)
	// Even legacy seconds in storage must not reorder or repeat minute-based dates.
	insertProjectedDate(t, h, 1, "2026-10-30", "19:30:59.999999", "")
	insertProjectedDate(t, h, 2, "2026-10-30", "19:30", "")
	params := url.Values{"start": {"2026-10-30"}, "limit": {"1"}}
	for _, id := range []int{1, 2} {
		data := getEventsPage(t, router, http.MethodGet, params)
		if len(data.Events) != 1 || data.Events[0].DateUuid != projectedDateUUID(id) || data.LastEventStartAt == nil || *data.LastEventStartAt != "2026-10-30T19:30" {
			t.Fatalf("minute-based sort and UUID tie-breaker lost: %+v", data)
		}
		params.Set("last_event_start_at", *data.LastEventStartAt)
		params.Set("last_event_date_uuid", *data.LastEventDateUuid)
	}
	if data := getEventsPage(t, router, http.MethodGet, params); len(data.Events) != 0 {
		t.Fatalf("completed minute must not repeat: %+v", data)
	}
}

func TestGetEventsPostgresOtherCursorQueries(t *testing.T) {
	h, router := eventsDatabase(t)
	insertProjectedDate(t, h, 1, "2026-10-01", "09:00", "2026-11-15")
	insertProjectedDate(t, h, 2, "2026-10-31", "19:00", "")
	params := url.Values{
		"start": {"2026-10-30"}, "last_event_start_at": {"2026-10-30T23:59"},
		"last_event_date_uuid": {projectedDateUUID(1)},
	}
	for _, path := range []string{"/api/events/type-summary", "/api/events/geojson"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path+"?"+params.Encode(), nil))
			if response.Code != http.StatusOK {
				t.Fatalf("cursor must work without the list's SQL alias: %d %s", response.Code, response.Body.String())
			}
			var payload struct {
				Data struct {
					Total    int `json:"total_event_count"`
					Features []struct {
						Properties struct {
							Count int `json:"event_count"`
						} `json:"properties"`
					} `json:"features"`
				} `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			total := payload.Data.Total
			if path == "/api/events/geojson" {
				for _, feature := range payload.Data.Features {
					total += feature.Properties.Count
				}
			}
			if total != 1 {
				t.Fatalf("original start filter must leave only tomorrow's date: %s", response.Body.String())
			}
		})
	}
}
