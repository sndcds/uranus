package api

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
	"github.com/sndcds/uranus/model"
)

func venueScopeRequest(handler gin.HandlerFunc, method, path, body string, params gin.Params) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(w)
	gc.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	gc.Request.Header.Set("Content-Type", "application/json")
	gc.Set("user-uuid", authTestUser)
	gc.Params = params
	handler(gc)
	return w
}

func TestVenueScopeCreateRejectsBeforeDatabase(t *testing.T) {
	h := setupAuthTest(t) // No pool: any database access would fail this test.
	for _, value := range []any{"standard", "foo", "", "shared'", "Organization", "SHARED", "  ", nil, 42} {
		body, err := json.Marshal(map[string]any{"org_uuid": authTestUser, "venue_name": "Venue", "scope": value})
		if err != nil {
			t.Fatal(err)
		}
		w := venueScopeRequest(h.AdminCreateVenue, "POST", "/", string(body), nil)
		if w.Code != 400 {
			t.Fatalf("%v: expected 400, got %d: %s", value, w.Code, w.Body)
		}
	}
	w := venueScopeRequest(h.AdminCreateVenue, "POST", "/", `{"venue_name":"Venue","org_uuid":"`+authTestUser+`"}`, nil)
	if w.Code != 400 {
		t.Fatal("missing scope was accepted")
	}
}

func TestVenueScopeGeoJSONRejectsBeforeDatabase(t *testing.T) {
	h := setupAuthTest(t)
	for _, scopes := range []string{"standard", "foo", "shared'", "Organization", "SHARED", " ", "organization,", ",shared", "organization,foo"} {
		w := venueScopeRequest(h.GetVenuesGeoJSON, "GET",
			"/?bbox=0,0,10,10&scopes="+url.QueryEscape(scopes), "", nil)
		if w.Code != 400 {
			t.Fatalf("%q: expected 400, got %d: %s", scopes, w.Code, w.Body)
		}
	}
}

func TestVenueScopeContracts(t *testing.T) {
	want := []string{string(model.VenueScopeOrganization), string(model.VenueScopeShared)}
	for _, file := range []string{"oas3-get-choosable-venues.yaml", "oas3-combined.yaml"} {
		data, err := os.ReadFile("../openapi/" + file)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Components struct {
				Schemas map[string]struct {
					Properties map[string]struct {
						Type string
						Enum []string
					}
				}
			}
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		scope := doc.Components.Schemas["Venue"].Properties["scope"]
		if scope.Type != "string" || !reflect.DeepEqual(scope.Enum, want) {
			t.Fatalf("%s: scope contract = %+v", file, scope)
		}
	}
	data, err := os.ReadFile("../ddl/venue.ddl")
	if err != nil {
		t.Fatal(err)
	}
	line := regexp.MustCompile(`(?m)^\s*scope text[^\n]*`).FindString(string(data))
	if strings.Contains(strings.ToUpper(line), "DEFAULT") || !strings.Contains(line, "NOT NULL") {
		t.Fatalf("scope must be required without a default: %s", line)
	}
	matches := regexp.MustCompile(`'([^']*)'`).FindAllStringSubmatch(line, -1)
	var values []string
	for _, match := range matches {
		values = append(values, match[1])
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("DDL values = %v, want %v", values, want)
	}
}

func TestVenueScopeLegacyUpsertIsNotRouted(t *testing.T) {
	// Parse Go rather than matching comments: reactivating the historical route
	// must not silently reintroduce its scope-less INSERT.
	file, err := parser.ParseFile(token.NewFileSet(), "../uranus-api.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	foundCreate := false
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if selector.Sel.Name == "AdminUpsertVenue" {
				t.Error("legacy upsert is routed; migrate its identity and scope contract before activation")
			}
			if selector.Sel.Name == "AdminCreateVenue" {
				foundCreate = true
			}
		}
		return true
	})
	if !foundCreate {
		t.Fatal("expected active AdminCreateVenue route")
	}
}
