package model

import (
	"encoding/json"
	"testing"
)

func TestVenueScope(t *testing.T) {
	for _, scope := range []VenueScope{VenueScopeOrganization, VenueScopeShared} {
		if !scope.IsValid() {
			t.Fatalf("valid scope rejected: %q", scope)
		}
		data, err := json.Marshal(Venue{Scope: scope})
		if err != nil {
			t.Fatal(err)
		}
		var response map[string]any
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		if response["scope"] != string(scope) {
			t.Fatalf("scope JSON contract changed: %s", data)
		}
	}
	for _, value := range []VenueScope{"", "standard", "foo", "shared'", "Organization", "SHARED", " organization "} {
		if value.IsValid() {
			t.Fatalf("invalid scope accepted: %q", value)
		}
	}
}
