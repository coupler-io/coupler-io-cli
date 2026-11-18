package integrations_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/commands/integrations"
)

func TestListSourcesAndDestinations(t *testing.T) {
	for kind, key := range map[string]string{"sources": "csv", "destinations": "google_spreadsheet"} {
		t.Run(kind, func(t *testing.T) {
			env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/integrations/"+kind {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				cmdtest.JSON(w, 200, map[string]interface{}{"integrations": []map[string]string{{"key": key, "name": "Name"}}})
			})

			if err := cmdtest.Run(integrations.NewCmdIntegrations(env.Factory), kind, "list"); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(env.Out.String(), key) {
				t.Errorf("missing %q in:\n%s", key, env.Out.String())
			}
		})
	}
}

func TestGetShowsParams(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/integrations/sources/csv" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{
			"key": "csv", "name": "CSV", "credential_provider": nil,
			"params": []map[string]interface{}{{"name": "url", "title": "File URL", "type": "string", "required": true, "helper_text": "Public link"}},
		})
	})

	if err := cmdtest.Run(integrations.NewCmdIntegrations(env.Factory), "sources", "get", "csv"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"CSV", "Params:", "url", "File URL", "true", "Public link"} {
		if !strings.Contains(env.Out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, env.Out.String())
		}
	}
}

func TestGetDestinationUsesDestinationsPath(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/integrations/destinations/google_spreadsheet" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"key": "google_spreadsheet", "name": "Google Sheets"})
	})

	if err := cmdtest.Run(integrations.NewCmdIntegrations(env.Factory), "destinations", "get", "google_spreadsheet"); err != nil {
		t.Fatal(err)
	}
}

// The API documents dependent values as params[<name>] (a Rails-style nested
// query). This pins the exact wire format so a refactor cannot change it.
func TestFieldOptionsQueryFormat(t *testing.T) {
	var rawQuery string
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces/42/integrations/sources/gsheet/field_options" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		rawQuery = r.URL.RawQuery
		cmdtest.JSON(w, 200, map[string]interface{}{"field": "sheetsList", "options": []string{"Sheet1"}})
	})

	err := cmdtest.Run(integrations.NewCmdIntegrations(env.Factory), "sources", "field-options", "gsheet",
		"--field", "sheetsList", "--credential-id", "15",
		"--param", "spreadsheetURL=https://x/a?b=c", "--param", "reportType.value=search_results_performance")
	if err != nil {
		t.Fatal(err)
	}

	q, err := url.ParseQuery(rawQuery)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"field":                    "sheetsList",
		"credential_id":            "15",
		"params[spreadsheetURL]":   "https://x/a?b=c",
		"params[reportType.value]": "search_results_performance",
	}
	if len(q) != len(want) {
		t.Errorf("unexpected query %v", q)
	}
	for key, value := range want {
		if q.Get(key) != value {
			t.Errorf("query[%q] = %q, want %q (raw: %s)", key, q.Get(key), value, rawQuery)
		}
	}
	if !strings.Contains(env.Out.String(), "Sheet1") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestFieldOptionsOmitsOptionalQuery(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "field=sheetsList" {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"options": []string{}})
	})

	if err := cmdtest.Run(integrations.NewCmdIntegrations(env.Factory), "sources", "field-options", "gsheet", "--field", "sheetsList"); err != nil {
		t.Fatal(err)
	}
}

func TestFieldOptionsUsageErrors(t *testing.T) {
	tests := map[string][]string{
		"missing field":    {"sources", "field-options", "gsheet"},
		"malformed param":  {"sources", "field-options", "gsheet", "--field", "f", "--param", "novalue"},
		"empty param name": {"sources", "field-options", "gsheet", "--field", "f", "--param", "=value"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
			err := cmdtest.Run(integrations.NewCmdIntegrations(env.Factory), args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			if code := cmdutil.ExitCode(err); code != cmdutil.ExitUsage {
				t.Errorf("exit code = %d, want %d (%v)", code, cmdutil.ExitUsage, err)
			}
		})
	}
}

func TestFieldOptionsAPIError(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 422, map[string]interface{}{"error": map[string]string{"message": "credential_id is required"}})
	})

	err := cmdtest.Run(integrations.NewCmdIntegrations(env.Factory), "sources", "field-options", "gsheet", "--field", "sheetsList")
	if err == nil || !strings.Contains(err.Error(), "credential_id is required") {
		t.Fatalf("expected the API message, got %v", err)
	}
}
