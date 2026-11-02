package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
)

func TestGetPortalGeometrySelectRejectsInvalidUUID(t *testing.T) {
	h := setupAuthTest(t)
	// A nil database pool ensures invalid input is rejected before any query.
	router := gin.New()
	router.GET("/api/portal/:uuid/geometries/select", h.GetPortalGeometrySelect)
	router.GET("/missing-uuid", h.GetPortalGeometrySelect)
	for _, path := range []string{
		"/missing-uuid",
		"/api/portal/not-a-uuid/geometries/select",
		"/api/portal/00000000-0000-0000-0000-000000000000/geometries/select",
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: got %d: %s", path, w.Code, w.Body.String())
		}
	}
}

func TestGetPortalGeometrySelectRejectsInvalidGroup(t *testing.T) {
	h := setupAuthTest(t)
	router := gin.New()
	router.GET("/api/portal/:uuid/geometries/select", h.GetPortalGeometrySelect)
	for _, query := range []string{
		"group", "group=", "group=abc", "group=1.5",
		"group=2147483648", "group=-2147483649",
	} {
		w := httptest.NewRecorder()
		path := "/api/portal/" + uuid.NewString() + "/geometries/select?" + query
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: got %d: %s", query, w.Code, w.Body.String())
		}
	}
}

func TestGetPortalGeometrySelectPostgres(t *testing.T) {
	dsn := os.Getenv("URANUS_PORTAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set URANUS_PORTAL_TEST_DATABASE_URL to run PostgreSQL portal geometry tests")
	}
	h := setupAuthTest(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h.DbPool = pool
	h.DbSchema = "portal_select_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	dbExec(t, h, "CREATE SCHEMA "+h.DbSchema)
	t.Cleanup(func() { dbExec(t, h, "DROP SCHEMA "+h.DbSchema+" CASCADE") })
	// Selection only needs metadata, so the fixture deliberately has no geometry.
	dbExec(t, h, `
		CREATE TABLE uranus.portal2 (uuid uuid PRIMARY KEY);
		CREATE TABLE uranus.portal_geometry (
			uuid uuid PRIMARY KEY,
			portal_uuid uuid REFERENCES uranus.portal2(uuid),
			parent_uuid uuid REFERENCES uranus.portal_geometry(uuid),
			name text NOT NULL,
			"order" integer DEFAULT 0,
			group_id integer,
			admin_level integer
		);
	`)
	query, err := os.ReadFile("../sql/get-portal-geometry-select.sql")
	if err != nil {
		t.Fatal(err)
	}
	app.UranusInstance.SqlGetPortalGeometrySelect = strings.ReplaceAll(string(query), "{{schema}}", h.DbSchema)
	router := gin.New()
	router.GET("/api/portal/:uuid/geometries/select", h.GetPortalGeometrySelect)
	portal, otherPortal, emptyPortal := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, id := range []string{portal, otherPortal, emptyPortal} {
		dbExec(t, h, "INSERT INTO uranus.portal2 (uuid) VALUES ($1)", id)
	}
	root, first, defaultOrder, alpha, tieA, tieB := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	if tieA > tieB {
		tieA, tieB = tieB, tieA
	}
	insertGeometry := func(id, portalID string, parentID any, name string, order, groupID, level any) {
		t.Helper()
		dbExec(t, h, `INSERT INTO uranus.portal_geometry
			(uuid, portal_uuid, parent_uuid, name, "order", group_id, admin_level)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, id, portalID, parentID, name, order, groupID, level)
	}
	// Insert out of display order; roots and descendants share one flat list.
	insertGeometry(root, portal, nil, "Aardvark", 2, 2, 6)
	insertGeometry(tieB, portal, root, "Beta", 1, 2, nil)
	insertGeometry(first, portal, root, "Zulu", -1, 0, 0)
	insertGeometry(tieA, portal, root, "Beta", 1, 2, nil)
	insertGeometry(alpha, portal, root, "Alpha", 1, 2, 8)
	insertGeometry(defaultOrder, portal, tieB, "Default", nil, nil, nil)
	insertGeometry(uuid.NewString(), otherPortal, nil, "Other portal", -10, 2, 10)

	request := func(t *testing.T, id, query string, wantStatus int) map[string]json.RawMessage {
		t.Helper()
		w := httptest.NewRecorder()
		path := "/api/portal/" + id + "/geometries/select"
		if query != "" {
			path += "?" + query
		}
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != wantStatus {
			t.Fatalf("got %d, want %d: %s", w.Code, wantStatus, w.Body.String())
		}
		var response struct {
			ResponseType string                     `json:"response_type"`
			Status       int                        `json:"status"`
			Metadata     map[string]json.RawMessage `json:"metadata"`
			Data         map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.ResponseType != "get-portal-geometry-select" || response.Status != wantStatus ||
			response.Metadata["response_time_ms"] == nil || string(response.Metadata["portal_uuid"]) != `"`+id+`"` {
			t.Fatalf("unexpected API response header: %s", w.Body.String())
		}
		return response.Data
	}

	t.Run("sorted flat selection and fields", func(t *testing.T) {
		data := request(t, portal, "", http.StatusOK)
		var items []map[string]any
		if err := json.Unmarshal(data["select"], &items); err != nil {
			t.Fatal(err)
		}
		want := []map[string]any{
			{"name": "Zulu", "order": float64(-1), "group_id": float64(0), "admin_level": float64(0), "geometry_uuid": first},
			{"name": "Default", "order": float64(0), "group_id": nil, "admin_level": nil, "geometry_uuid": defaultOrder},
			{"name": "Alpha", "order": float64(1), "group_id": float64(2), "admin_level": float64(8), "geometry_uuid": alpha},
			{"name": "Beta", "order": float64(1), "group_id": float64(2), "admin_level": nil, "geometry_uuid": tieA},
			{"name": "Beta", "order": float64(1), "group_id": float64(2), "admin_level": nil, "geometry_uuid": tieB},
			{"name": "Aardvark", "order": float64(2), "group_id": float64(2), "admin_level": float64(6), "geometry_uuid": root},
		}
		if len(data) != 1 || !reflect.DeepEqual(items, want) {
			t.Fatalf("unexpected selection: %s", data["select"])
		}
	})
	t.Run("group filter preserves ordering and portal scope", func(t *testing.T) {
		data := request(t, portal, "group=2", http.StatusOK)
		var items []portalGeometrySelectItem
		if err := json.Unmarshal(data["select"], &items); err != nil {
			t.Fatal(err)
		}
		want := []string{alpha, tieA, tieB, root}
		if len(items) != len(want) {
			t.Fatalf("unexpected group selection: %s", data["select"])
		}
		for i, item := range items {
			if item.GeometryUuid != want[i] || item.GroupID == nil || *item.GroupID != 2 {
				t.Fatalf("unexpected group selection: %s", data["select"])
			}
		}
	})
	t.Run("group zero", func(t *testing.T) {
		data := request(t, portal, "group=0", http.StatusOK)
		var items []portalGeometrySelectItem
		if err := json.Unmarshal(data["select"], &items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].GeometryUuid != first || items[0].GroupID == nil || *items[0].GroupID != 0 {
			t.Fatalf("unexpected group zero selection: %s", data["select"])
		}
	})
	t.Run("no matching group", func(t *testing.T) {
		if data := request(t, portal, "group=99", http.StatusOK); string(data["select"]) != "[]" {
			t.Fatalf("expected select: [], got %s", data["select"])
		}
	})
	t.Run("no geometries", func(t *testing.T) {
		for _, query := range []string{"", "group=2"} {
			if data := request(t, emptyPortal, query, http.StatusOK); string(data["select"]) != "[]" {
				t.Fatalf("expected select: [], got %s", data["select"])
			}
		}
	})
	t.Run("unknown portal", func(t *testing.T) {
		for _, query := range []string{"", "group=2"} {
			request(t, uuid.NewString(), query, http.StatusNotFound)
		}
	})
	t.Run("database error", func(t *testing.T) {
		dbExec(t, h, "DROP TABLE uranus.portal_geometry")
		request(t, portal, "", http.StatusInternalServerError)
	})
}
