package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sndcds/uranus/app"
)

func TestGetPortalGeometriesRejectsInvalidUUID(t *testing.T) {
	h := setupAuthTest(t)
	// A nil database pool ensures invalid input is rejected before any query.
	router := gin.New()
	router.GET("/api/portal/:uuid/geometries", h.GetPortalGeometries)
	router.GET("/missing-uuid", h.GetPortalGeometries)
	for _, path := range []string{
		"/missing-uuid",
		"/api/portal/not-a-uuid/geometries",
		"/api/portal/00000000-0000-0000-0000-000000000000/geometries",
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: got %d: %s", path, w.Code, w.Body.String())
		}
	}
}

func TestGetPortalGeometriesPostgres(t *testing.T) {
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
	h.DbSchema = "portal_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	dbExec(t, h, "CREATE SCHEMA "+h.DbSchema)
	t.Cleanup(func() { dbExec(t, h, "DROP SCHEMA "+h.DbSchema+" CASCADE") })
	// Only the columns used by the endpoint are needed; no PostGIS operations
	// or polygon coordinates are involved in the selection hierarchy.
	dbExec(t, h, `
		CREATE TABLE uranus.portal2 (uuid uuid PRIMARY KEY);
		CREATE TABLE uranus.portal_geometry (
			uuid uuid PRIMARY KEY,
			portal_uuid uuid REFERENCES uranus.portal2(uuid),
			parent_uuid uuid REFERENCES uranus.portal_geometry(uuid),
			name text NOT NULL,
			description text NOT NULL,
			"order" integer DEFAULT 0,
			admin_level integer,
			type text
		);
	`)
	query, err := os.ReadFile("../sql/get-portal-geometries.sql")
	if err != nil {
		t.Fatal(err)
	}
	app.UranusInstance.SqlGetPortalGeometries = strings.ReplaceAll(string(query), "{{schema}}", h.DbSchema)
	router := gin.New()
	router.GET("/api/portal/:uuid/geometries", h.GetPortalGeometries)
	portal, otherPortal, emptyPortal := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, id := range []string{portal, otherPortal, emptyPortal} {
		dbExec(t, h, "INSERT INTO uranus.portal2 (uuid) VALUES ($1)", id)
	}
	root, child, grandchild, first, tieA, tieB := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	if tieA > tieB {
		tieA, tieB = tieB, tieA
	}
	insertGeometry := func(id, portalID string, parentID *string, name string, order *int, level *int, typ *string) {
		t.Helper()
		dbExec(t, h, `INSERT INTO uranus.portal_geometry
			(uuid, portal_uuid, parent_uuid, name, description, "order", admin_level, type)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, id, portalID, parentID, name, name+" description", order, level, typ)
	}
	rootOrder, firstOrder, childOrder, level, typ := 100, -1, 2, 8, "city"
	insertGeometry(root, portal, nil, "City", &rootOrder, &level, &typ)
	// Insert out of display order, including rows sorted before their parents.
	insertGeometry(child, portal, &root, "District", &childOrder, nil, nil)
	insertGeometry(grandchild, portal, &child, "Quarter", nil, nil, nil)
	insertGeometry(tieB, portal, &root, "Same name", nil, nil, nil)
	insertGeometry(first, portal, &root, "First", &firstOrder, nil, nil)
	insertGeometry(tieA, portal, &root, "Same name", nil, nil, nil)
	insertGeometry(uuid.NewString(), otherPortal, nil, "Other portal", nil, nil, nil)

	request := func(id string, wantStatus int) map[string]json.RawMessage {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/portal/"+id+"/geometries", nil))
		if w.Code != wantStatus {
			t.Fatalf("got %d, want %d: %s", w.Code, wantStatus, w.Body.String())
		}
		var response struct {
			Service      string                     `json:"service"`
			APIVersion   string                     `json:"api_version"`
			ResponseType string                     `json:"response_type"`
			Status       int                        `json:"status"`
			Timestamp    string                     `json:"timestamp"`
			Metadata     map[string]json.RawMessage `json:"metadata"`
			Data         map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Service == "" || response.APIVersion == "" || response.Timestamp == "" ||
			response.ResponseType != "get-portal-geometries" || response.Status != wantStatus ||
			response.Metadata["response_time_ms"] == nil || string(response.Metadata["portal_uuid"]) != `"`+id+`"` {
			t.Fatalf("missing API response header: %s", w.Body.String())
		}
		return response.Data
	}

	t.Run("hierarchy and ordering", func(t *testing.T) {
		data := request(portal, http.StatusOK)
		var geometry portalGeometry
		if err := json.Unmarshal(data["geometry"], &geometry); err != nil {
			t.Fatal(err)
		}
		if geometry.Uuid != root || geometry.Name != "City" || geometry.Description != "City description" ||
			geometry.Order != rootOrder || geometry.AdminLevel == nil || *geometry.AdminLevel != level ||
			geometry.Type == nil || *geometry.Type != typ || len(geometry.Childs) != 4 {
			t.Fatalf("unexpected main geometry: %s", data["geometry"])
		}
		for i, want := range []string{first, tieA, tieB, child} {
			if geometry.Childs[i].Uuid != want {
				t.Fatalf("child %d: got %s, want %s", i, geometry.Childs[i].Uuid, want)
			}
		}
		quarter := geometry.Childs[3].Childs
		if len(quarter) != 1 || quarter[0].Uuid != grandchild || quarter[0].Childs == nil || len(quarter[0].Childs) != 0 {
			t.Fatalf("unexpected nested geometry: %s", data["geometry"])
		}
		if geometry.Childs[1].Order != 0 || geometry.Childs[1].Type != nil || geometry.Childs[1].AdminLevel != nil ||
			strings.Contains(string(data["geometry"]), "parent_uuid") {
			t.Fatalf("unexpected nullable fields or internal fields: %s", data["geometry"])
		}
	})
	t.Run("no geometries", func(t *testing.T) {
		if data := request(emptyPortal, http.StatusOK); string(data["geometry"]) != "null" {
			t.Fatalf("expected geometry: null, got %s", data["geometry"])
		}
	})
	t.Run("unknown portal", func(t *testing.T) {
		request(uuid.NewString(), http.StatusNotFound)
	})
	t.Run("multiple roots", func(t *testing.T) {
		id := uuid.NewString()
		insertGeometry(id, portal, nil, "Second root", nil, nil, nil)
		request(portal, http.StatusInternalServerError)
		dbExec(t, h, "DELETE FROM uranus.portal_geometry WHERE uuid=$1", id)
	})
	t.Run("parent belongs to another portal", func(t *testing.T) {
		id := uuid.NewString()
		insertGeometry(id, otherPortal, &root, "Foreign child", nil, nil, nil)
		request(portal, http.StatusOK)
		request(otherPortal, http.StatusInternalServerError)
		dbExec(t, h, "DELETE FROM uranus.portal_geometry WHERE uuid=$1", id)
	})
	t.Run("cycle", func(t *testing.T) {
		dbExec(t, h, "UPDATE uranus.portal_geometry SET parent_uuid=$1 WHERE uuid=$2", grandchild, child)
		request(portal, http.StatusInternalServerError)
		dbExec(t, h, "UPDATE uranus.portal_geometry SET parent_uuid=$1 WHERE uuid=$2", root, child)
	})
	t.Run("missing root", func(t *testing.T) {
		dbExec(t, h, "UPDATE uranus.portal_geometry SET parent_uuid=uuid WHERE uuid=$1", root)
		request(portal, http.StatusInternalServerError)
	})
}
