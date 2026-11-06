package sources_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/commands/sources"
)

func TestCreate(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/workspaces/42/dataflows/df-1/sources" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := map[string]interface{}{
			"integration_key": "csv",
			"credential_id":   float64(15),
			"params":          map[string]interface{}{"url": "https://example.com/data.csv", "method": "GET"},
		}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v\nwant %v", body, want)
		}
		cmdtest.JSON(w, 201, map[string]interface{}{
			"id": "s-1", "name": "CSV", "integration": map[string]string{"key": "csv"},
			"enabled": true, "lifecycle_state": "active", "params_configured": true,
			"params_remaining_errors": map[string]interface{}{},
		})
	})

	if err := cmdtest.Run(sources.NewCmdSources(env.Factory), "create", "--dataflow-id", "df-1", "--integration", "csv", "--credential-id", "15",
		"--params", `{"method":"GET"}`, "--param", "url=https://example.com/data.csv"); err != nil {
		t.Fatal(err)
	}
	out := env.Out.String()
	for _, want := range []string{"s-1", "csv", "active", "true"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestCreateOmitsUnsetOptionalFields(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, map[string]interface{}{"integration_key": "csv"}) {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 201, map[string]interface{}{"id": "s-1"})
	})

	if err := cmdtest.Run(sources.NewCmdSources(env.Factory), "create", "--dataflow-id", "df-1", "--integration", "csv"); err != nil {
		t.Fatal(err)
	}
}

func TestCreateRequiredFlags(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })

	err := cmdtest.Run(sources.NewCmdSources(env.Factory), "create", "--integration", "csv")
	if err == nil || !strings.Contains(err.Error(), "--dataflow-id is required") {
		t.Fatalf("got %v", err)
	}
	err = cmdtest.Run(sources.NewCmdSources(env.Factory), "create", "--dataflow-id", "df-1")
	if err == nil || !strings.Contains(err.Error(), "--integration is required") {
		t.Fatalf("got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/workspaces/42/sources/s-1" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := map[string]interface{}{
			"enabled":         false,
			"lifecycle_state": "paused",
			"params":          map[string]interface{}{"url": "https://example.com/new.csv"},
		}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v\nwant %v", body, want)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"id": "s-1", "name": "CSV", "enabled": false, "lifecycle_state": "paused"})
	})

	err := cmdtest.Run(sources.NewCmdSources(env.Factory), "update", "s-1",
		"--enabled=false", "--lifecycle-state", "paused", "--param", "url=https://example.com/new.csv")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "paused") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestUpdateValidation(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })

	err := cmdtest.Run(sources.NewCmdSources(env.Factory), "update", "s-1")
	if err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("got %v", err)
	}
	err = cmdtest.Run(sources.NewCmdSources(env.Factory), "update", "s-1", "--lifecycle-state", "asleep")
	if err == nil || !strings.Contains(err.Error(), "--lifecycle-state must be one of") {
		t.Fatalf("got %v", err)
	}
}
