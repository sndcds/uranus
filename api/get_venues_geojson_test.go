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
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
)

func TestGetVenuesGeoJSONPostgresPortalGeometry(t *testing.T) {
	dsn := os.Getenv("URANUS_VENUES_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set URANUS_VENUES_TEST_DATABASE_URL to run PostgreSQL venue geometry tests")
	}
	h := setupAuthTest(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h.DbPool = pool
	h.DbSchema = "venues_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := pool.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS postgis"); err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, "CREATE SCHEMA "+h.DbSchema)
	t.Cleanup(func() { dbExec(t, h, "DROP SCHEMA "+h.DbSchema+" CASCADE") })
	fixture, err := os.ReadFile("testdata/get_venues_geojson.sql")
	if err != nil {
		t.Fatal(err)
	}
	dbExec(t, h, string(fixture))
	for path, destination := range map[string]*string{
		"get-venues-geojson.sql":        &app.UranusInstance.SqlGetVenuesGeoJSON,
		"get-portal-venues-geojson.sql": &app.UranusInstance.SqlGetPortalVenuesGeoJSON,
	} {
		query, err := os.ReadFile("../sql/" + path)
		if err != nil {
			t.Fatal(err)
		}
		*destination = strings.ReplaceAll(string(query), "{{schema}}", h.DbSchema)
	}
	router := gin.New()
	router.GET("/api/venues/geojson", h.GetVenuesGeoJSON)
	testUUID := func(id int) string {
		return fmt.Sprintf("00000000-0000-0000-0000-%012d", id)
	}
	for _, tc := range []struct {
		name, geographicFilter, orgFilter string
		geometry, portal                  int
		scopes, bbox                      string
		want                              []int
	}{
		{name: "main geometry", want: []int{31, 32, 33, 34, 37, 38}},
		{name: "selected child geometry", geometry: 12, want: []int{31, 33, 37}},
		{name: "unknown geometry does not fall back", geometry: 99},
		{name: "missing main geometry", portal: 2},
		{name: "scope with main geometry", scopes: "organization", want: []int{32}},
		{name: "scope with child geometry", geometry: 12, scopes: "organization"},
		{name: "bbox with child geometry", geometry: 12, bbox: "1,1,4,4", want: []int{31, 33}},
		{name: "allowlist with main geometry", orgFilter: "allowlist", want: []int{31, 32, 37}},
		{name: "allowlist with child geometry", geometry: 12, orgFilter: "allowlist", want: []int{31, 37}},
		{name: "blocklist with child geometry", geometry: 12, orgFilter: "blocklist", want: []int{31, 37}},
		{name: "allowlist without geographic restriction", geographicFilter: "none", orgFilter: "allowlist", want: []int{31, 32, 35, 37}},
		{name: "blocklist without geographic restriction", geographicFilter: "none", orgFilter: "blocklist", want: []int{31, 32, 34, 35, 37, 38}},
		{name: "no geographic restriction ignores geometry", geographicFilter: "none", geometry: 99, want: []int{31, 32, 33, 34, 35, 37, 38}},
		{name: "no restriction needs no main geometry", geographicFilter: "none", portal: 2, want: []int{31, 32, 33, 34, 35, 37, 38}},
		{name: "without portal", portal: -1, want: []int{31, 32, 33, 34, 35, 37, 38}},
		{name: "scope without portal", portal: -1, scopes: "organization", want: []int{32}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.geographicFilter == "" {
				tc.geographicFilter = "geometry"
			}
			if tc.orgFilter == "" {
				tc.orgFilter = "all"
			}
			dbExec(t, h, "UPDATE uranus.portal2 SET geographic_filter=$1, org_filter=$2", tc.geographicFilter, tc.orgFilter)
			params := url.Values{"bbox": {"-1,-1,15,15"}}
			if tc.portal >= 0 {
				portal := tc.portal
				if portal == 0 {
					portal = 1
				}
				params.Set("portal", testUUID(portal))
			}
			if tc.geometry != 0 {
				params.Set("geometry", testUUID(tc.geometry))
			}
			if tc.scopes != "" {
				params.Set("scopes", tc.scopes)
			}
			if tc.bbox != "" {
				params.Set("bbox", tc.bbox)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/venues/geojson?"+params.Encode(), nil))
			wantStatus := http.StatusOK
			if len(tc.want) == 0 {
				wantStatus = http.StatusNoContent
			}
			if response.Code != wantStatus {
				t.Fatalf("got status %d, want %d: %s", response.Code, wantStatus, response.Body.String())
			}
			if wantStatus == http.StatusNoContent {
				return
			}
			var payload struct {
				Data struct {
					Type     string `json:"type"`
					Features []struct {
						Properties struct {
							UUID string `json:"uuid"`
						} `json:"properties"`
					} `json:"features"`
				} `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Data.Type != "FeatureCollection" {
				t.Fatalf("unexpected GeoJSON type: %q", payload.Data.Type)
			}
			var got, want []string
			for _, feature := range payload.Data.Features {
				got = append(got, feature.Properties.UUID)
			}
			for _, id := range tc.want {
				want = append(want, testUUID(id))
			}
			sort.Strings(got)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got venues %v, want %v", got, want)
			}
		})
	}
}
