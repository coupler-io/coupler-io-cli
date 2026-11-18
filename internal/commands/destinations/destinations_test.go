package destinations_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/commands/destinations"
)

func TestCreate(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/workspaces/42/dataflows/df-1/destinations" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := map[string]interface{}{
			"integration_key": "google_spreadsheet",
			"credential_id":   float64(15),
			"params":          map[string]interface{}{"sheet_name": "Sales"},
		}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v\nwant %v", body, want)
		}
		cmdtest.JSON(w, 201, map[string]interface{}{
			"id": "d-1", "name": "Sales sheet", "integration": map[string]string{"key": "google_spreadsheet"},
			"enabled": true, "lifecycle_state": "active",
		})
	})

	err := cmdtest.Run(destinations.NewCmdDestinations(env.Factory), "create", "--dataflow-id", "df-1",
		"--integration", "google_spreadsheet", "--credential-id", "15", "--param", "sheet_name=Sales")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "google_spreadsheet") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestUpdate(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/workspaces/42/destinations/d-1" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := map[string]interface{}{"name": "Q4", "lifecycle_state": "paused"}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := cmdtest.Run(destinations.NewCmdDestinations(env.Factory), "update", "d-1", "--name", "Q4", "--lifecycle-state", "paused"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "Destination d-1 updated.") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

// The API's destination PATCH has no `enabled` field, so the CLI must not
// offer a flag that would be silently ignored. Use --lifecycle-state disabled.
func TestUpdateHasNoEnabledFlag(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })

	err := cmdtest.Run(destinations.NewCmdDestinations(env.Factory), "update", "d-1", "--enabled=false")
	if err == nil || !strings.Contains(err.Error(), "unknown flag: --enabled") {
		t.Fatalf("got %v", err)
	}
}
