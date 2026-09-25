package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/grains/grains_uuid"
	"github.com/sndcds/uranus/app"
	"github.com/sndcds/uranus/model"
)

const venueScopeTestOrg = "01994126-6680-7000-8000-000000000002"

func venueScopeDatabase(t *testing.T) *ApiHandler {
	t.Helper()
	dsn := os.Getenv("URANUS_VENUE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set URANUS_VENUE_TEST_DATABASE_URL to a disposable PostgreSQL/PostGIS database")
	}
	h := setupAuthTest(t)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	id, err := grains_uuid.Uuidv7String()
	if err != nil {
		t.Fatal(err)
	}
	h.DbPool, h.DbSchema = pool, "venue_test_"+strings.ReplaceAll(id, "-", "")
	dbExec(t, h, "CREATE SCHEMA "+h.DbSchema)
	t.Cleanup(func() { dbExec(t, h, "DROP SCHEMA "+h.DbSchema+" CASCADE") })
	dbExec(t, h, "CREATE EXTENSION IF NOT EXISTS postgis")
	// Actual table definitions; omit snapshot-only duplicate indexes/triggers.
	for _, table := range []string{"user", "organization", "user_organization_link", "organization_member_link", "venue"} {
		data, err := os.ReadFile("../ddl/" + table + ".ddl")
		if err != nil {
			t.Fatal(err)
		}
		dbExec(t, h, strings.Split(string(data), "-- Indices")[0])
	}
	dbExec(t, h, `INSERT INTO uranus."user" (uuid,email,password_hash) VALUES ($1,'venue@example.test','unused')`, authTestUser)
	dbExec(t, h, "INSERT INTO uranus.organization (uuid,name) VALUES ($1,'Venue org')", venueScopeTestOrg)
	dbExec(t, h, "INSERT INTO uranus.user_organization_link (user_uuid,org_uuid,permissions) VALUES ($1,$2,$3)", authTestUser, venueScopeTestOrg, int64(app.UserPermCombinationAdmin))
	dbExec(t, h, "INSERT INTO uranus.organization_member_link (user_uuid,org_uuid,has_joined) VALUES ($1,$2,true)", authTestUser, venueScopeTestOrg)
	app.UranusInstance.SqlGetUserOrgPermissions = venueScopeSQL(t, h, "admin-get-user-org-permissions")
	return h
}

func venueScopeSQL(t *testing.T, h *ApiHandler, name string) string {
	t.Helper()
	data, err := os.ReadFile("../sql/" + name + ".sql")
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(strings.ReplaceAll(string(data), "{{schema}}", h.DbSchema), "{{base_api_url}}", "https://api.example.test")
}

func runVenueScopeMigration(t *testing.T, h *ApiHandler, direction string) error {
	t.Helper()
	data, err := os.ReadFile("../migrations/202609250002_venue_scope." + direction + ".sql")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	conn, err := h.DbPool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	_, err = conn.Exec(ctx, strings.ReplaceAll(string(data), "uranus.", h.DbSchema+"."))
	if err != nil {
		// A deliberately failing migration leaves its connection in an aborted
		// transaction. Roll back on that same connection before checking state.
		_, _ = conn.Exec(ctx, "ROLLBACK")
	}
	return err
}

func venueScopeReadFixtures(t *testing.T, h *ApiHandler) {
	t.Helper()
	// Minimal empty dependent tables; the venue/organization tables above and
	// every SELECT exercised below are the actual repository definitions.
	dbExec(t, h, `
		CREATE TABLE uranus.venue_type (key text, marker_style jsonb);
		CREATE TABLE uranus.space (uuid uuid, venue_uuid uuid, name text);
		CREATE TABLE uranus.event (uuid uuid, venue_uuid uuid, space_uuid uuid, release_status text);
		CREATE TABLE uranus.event_date (uuid uuid, event_uuid uuid, space_uuid uuid, start_date date);
		CREATE TABLE uranus.user_venue_link (venue_uuid uuid, user_uuid uuid, permissions bigint);
		CREATE TABLE uranus.organization_access_grants (src_org_uuid uuid, dst_org_uuid uuid, permissions bigint);
		CREATE TABLE uranus.pluto_image_link (context text, context_uuid uuid, identifier text, pluto_image_uuid uuid);
		CREATE TABLE uranus.pluto_image (uuid uuid, focus_x real, focus_y real, alt_text text,
			copyright text, license text, creator_name text);
		CREATE TABLE uranus.portal2 (uuid uuid, filter_type text, geometry geometry);
		CREATE TABLE uranus.portal_org_allowlist (portal_uuid uuid, org_uuid uuid);
		CREATE TABLE uranus.portal_org_blocklist (portal_uuid uuid, org_uuid uuid);
	`)
	dbExec(t, h, `INSERT INTO uranus.portal2 VALUES ($1,'allowlist',NULL)`, authTestUser)
	dbExec(t, h, "INSERT INTO uranus.portal_org_allowlist VALUES ($1,$2)", authTestUser, venueScopeTestOrg)
	a := app.UranusInstance
	a.SqlAdminGetVenue = venueScopeSQL(t, h, "admin-get-venue")
	a.SqlAdminGetOrgVenues = venueScopeSQL(t, h, "admin-get-org-venues")
	a.SqlAdminChoosableVenues = venueScopeSQL(t, h, "admin-chooseable-venues")
	a.SqlGetVenuesGeoJSON = venueScopeSQL(t, h, "get-venues-geojson")
	a.SqlGetPortalVenuesGeoJSON = venueScopeSQL(t, h, "get-portal-venues-geojson")
}

func TestVenueScopePostgresCreateAndRead(t *testing.T) {
	h := venueScopeDatabase(t)
	if err := runVenueScopeMigration(t, h, "up"); err != nil {
		t.Fatal(err)
	}
	venueScopeReadFixtures(t, h)
	for _, scope := range []model.VenueScope{model.VenueScopeOrganization, model.VenueScopeShared} {
		for _, padded := range []bool{false, true} {
			input := string(scope)
			if padded {
				input = " " + input + " " // Preserve the pre-existing create normalization.
			}
			body, _ := json.Marshal(map[string]string{"org_uuid": venueScopeTestOrg, "venue_name": "Test", "scope": input})
			w := venueScopeRequest(h.AdminCreateVenue, "POST", "/", string(body), nil)
			if w.Code != 201 {
				t.Fatalf("create %q: %d %s", input, w.Code, w.Body)
			}
			var created struct {
				Metadata struct {
					VenueUUID string `json:"venue_uuid"`
				} `json:"metadata"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
				t.Fatal(err)
			}
			id := created.Metadata.VenueUUID
			var stored model.VenueScope
			if err := h.DbPool.QueryRow(context.Background(), "SELECT scope FROM "+h.DbSchema+".venue WHERE uuid=$1", id).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored != scope {
				t.Fatalf("stored scope = %q, want %q", stored, scope)
			}
			dbExec(t, h, "UPDATE uranus.venue SET point=ST_SetSRID(ST_MakePoint(5,5),4326) WHERE uuid=$1", id)
			for _, route := range []struct {
				name    string
				handler gin.HandlerFunc
				params  gin.Params
			}{
				{"admin detail", h.AdminGetVenue, gin.Params{{Key: "venueUuid", Value: id}}},
				{"org list", h.AdminGetOrgVenues, gin.Params{{Key: "orgUuid", Value: venueScopeTestOrg}}},
				{"org choosable", h.AdminGetOrgChoosableVenues, gin.Params{{Key: "orgUuid", Value: venueScopeTestOrg}}},
				{"public choosable", h.GetChoosableVenues, nil},
			} {
				w := venueScopeRequest(route.handler, "GET", "/", "", route.params)
				if w.Code != 200 || !strings.Contains(w.Body.String(), `"scope":"`+string(scope)+`"`) {
					t.Fatalf("%s: scope missing in %d %s", route.name, w.Code, w.Body)
				}
			}
		}
	}
	for _, scopes := range []string{
		string(model.VenueScopeOrganization), string(model.VenueScopeShared),
		string(model.VenueScopeOrganization) + "," + string(model.VenueScopeShared),
		" organization , shared ", "",
	} {
		for _, portal := range []string{"", "&portal=" + authTestUser} {
			w := venueScopeRequest(h.GetVenuesGeoJSON, "GET", "/?bbox=0,0,10,10&scopes="+url.QueryEscape(scopes)+portal, "", nil)
			if w.Code != 200 {
				t.Fatalf("geojson %q: %d %s", scopes, w.Code, w.Body)
			}
			var response struct {
				Data struct {
					Features []struct {
						Properties struct {
							Scope model.VenueScope `json:"scope"`
						} `json:"properties"`
					} `json:"features"`
				} `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			want := 4
			if model.VenueScope(scopes).IsValid() {
				want = 2
			}
			if len(response.Data.Features) != want {
				t.Fatalf("filter %q returned %d features, want %d", scopes, len(response.Data.Features), want)
			}
			for _, feature := range response.Data.Features {
				if !feature.Properties.Scope.IsValid() || (want == 2 && string(feature.Properties.Scope) != scopes) {
					t.Fatalf("wrong scope in GeoJSON: %q", feature.Properties.Scope)
				}
			}
		}
	}
}

func TestVenueScopePostgresMigration(t *testing.T) {
	h := venueScopeDatabase(t)
	ctx := context.Background()
	// Reproduce the exact inconsistent default in the old DDL.
	dbExec(t, h, "ALTER TABLE uranus.venue ALTER COLUMN scope SET DEFAULT 'standard'")
	insert := "INSERT INTO " + h.DbSchema + ".venue (uuid,org_uuid,name) VALUES ($1,$2,'Test')"
	if _, err := h.DbPool.Exec(ctx, insert, authTestUser, venueScopeTestOrg); err == nil {
		t.Fatal("old default unexpectedly satisfied the CHECK")
	}
	dbExec(t, h, "INSERT INTO uranus.venue (uuid,org_uuid,name,scope) VALUES ($1,$2,'Test',$3)", authTestUser, venueScopeTestOrg, model.VenueScopeOrganization)
	if err := runVenueScopeMigration(t, h, "up"); err != nil {
		t.Fatal(err)
	}
	var def *string
	var nullable string
	err := h.DbPool.QueryRow(ctx, `SELECT column_default,is_nullable FROM information_schema.columns
		WHERE table_schema=$1 AND table_name='venue' AND column_name='scope'`, h.DbSchema).Scan(&def, &nullable)
	if err != nil || def != nil || nullable != "NO" {
		t.Fatalf("incorrect migrated column: %v, %v, %s", err, def, nullable)
	}
	// Missing scope must now fail NOT NULL, instead of using a fabricated default.
	if _, err := h.DbPool.Exec(ctx, insert, venueScopeTestOrg, venueScopeTestOrg); err == nil {
		t.Fatal("scope-less insert succeeded after migration")
	}
	for _, value := range []any{nil, "", "standard", "foo", "SHARED", "Organization", "shared'", " organization "} {
		if _, err := h.DbPool.Exec(ctx, "UPDATE "+h.DbSchema+".venue SET scope=$1", value); err == nil {
			t.Fatalf("database accepted invalid scope: %v", value)
		}
	}
	// A second migration is harmless; direct DB classification remains possible.
	if err := runVenueScopeMigration(t, h, "up"); err != nil {
		t.Fatal(err)
	}
	for _, value := range []model.VenueScope{model.VenueScopeShared, model.VenueScopeOrganization} {
		dbExec(t, h, "UPDATE uranus.venue SET scope=$1", value)
	}
	if err := runVenueScopeMigration(t, h, "down"); err == nil {
		t.Fatal("rollback must not silently restore an invalid default")
	}
}

func TestVenueScopePostgresMigrationRefusesInvalidData(t *testing.T) {
	for _, value := range []any{"standard", "foo", nil} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			h := venueScopeDatabase(t)
			dbExec(t, h, `ALTER TABLE uranus.venue DROP CONSTRAINT venue_scope_check,
				ALTER COLUMN scope DROP NOT NULL, ALTER COLUMN scope SET DEFAULT 'standard'`)
			dbExec(t, h, "INSERT INTO uranus.venue (uuid,org_uuid,name,scope) VALUES ($1,$2,'Legacy',$3)", authTestUser, venueScopeTestOrg, value)
			dbExec(t, h, `ALTER TABLE uranus.venue ADD CONSTRAINT venue_scope_check CHECK (scope IN ('organization','shared')) NOT VALID`)
			if err := runVenueScopeMigration(t, h, "up"); err == nil || !strings.Contains(err.Error(), "Invalid venue.scope") {
				t.Fatalf("migration should reject legacy data: %v", err)
			}
			var stored, def *string
			var validated bool
			err := h.DbPool.QueryRow(context.Background(), "SELECT scope FROM "+h.DbSchema+".venue WHERE uuid=$1", authTestUser).Scan(&stored)
			if err != nil || (value == nil && stored != nil) || (value != nil && (stored == nil || *stored != value.(string))) {
				t.Fatal("failed migration changed the legacy row")
			}
			err = h.DbPool.QueryRow(context.Background(), `SELECT column_default FROM information_schema.columns
				WHERE table_schema=$1 AND table_name='venue' AND column_name='scope'`, h.DbSchema).Scan(&def)
			if err != nil || def == nil || !strings.Contains(*def, "standard") {
				t.Fatal("failed migration changed the default")
			}
			err = h.DbPool.QueryRow(context.Background(), `SELECT convalidated FROM pg_constraint
				WHERE conrelid=$1::regclass AND conname='venue_scope_check'`, h.DbSchema+".venue").Scan(&validated)
			if err != nil || validated {
				t.Fatal("failed migration changed the original constraint")
			}
		})
	}
}
