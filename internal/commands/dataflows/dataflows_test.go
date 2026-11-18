package dataflows_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/commands/dataflows"
)

func TestList(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/workspaces/42/dataflows" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		cmdtest.JSON(w, 200, []map[string]interface{}{
			{"id": "df-1", "name": "Sales", "last_success_run_at": "2026-11-01T10:00:00Z", "last_successful_execution_id": "ex-1"},
			{"id": "df-2", "name": "Ads", "last_success_run_at": nil, "last_successful_execution_id": nil},
		})
	})

	if err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "list"); err != nil {
		t.Fatal(err)
	}
	out := env.Out.String()
	for _, want := range []string{"ID", "df-1", "Sales", "ex-1", "df-2", "Ads"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestListRequiresWorkspace(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request expected")
	})
	// Clearing the workspace must fail before any HTTP call.
	env.Factory.Viper.Set("workspace-id", "")

	err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "list")
	if err == nil || !strings.Contains(err.Error(), "workspace-id is required") {
		t.Fatalf("expected workspace error, got %v", err)
	}
}

func TestGetShowsSources(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces/42/dataflows/df-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{
			"id": "df-1", "name": "Sales", "schedule": "0 8 * * *", "last_successful_execution_id": "ex-1",
			"sources": []map[string]interface{}{
				{"id": "s-1", "name": "CSV", "type": "csv", "enabled": true, "lifecycle_state": "active", "params_configured": true},
			},
		})
	})

	if err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "get", "df-1"); err != nil {
		t.Fatal(err)
	}
	out := env.Out.String()
	for _, want := range []string{"Sales", "0 8 * * *", "Sources:", "s-1", "csv", "active"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestGetWithoutSources(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, map[string]interface{}{"id": "df-1", "name": "Empty", "sources": []interface{}{}})
	})

	if err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "get", "df-1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "(none)") {
		t.Errorf("expected (none) for empty sources:\n%s", env.Out.String())
	}
}

func TestCreate(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/workspaces/42/dataflows" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body := cmdtest.DecodeBody(t, r)
		if !reflect.DeepEqual(body, map[string]interface{}{"name": "Sales report"}) {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 201, map[string]interface{}{"id": "df-9", "name": "Sales report", "url": "https://app.coupler.io/app/flows/df-9"})
	})

	if err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "create", "--name", "Sales report"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "df-9") || !strings.Contains(env.Out.String(), "https://app.coupler.io/app/flows/df-9") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestCreateFromTemplate(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		body := cmdtest.DecodeBody(t, r)
		if body["template_id"] != "tpl-1" {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 201, map[string]interface{}{"id": "df-9", "name": "From template"})
	})

	if err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "create", "--template-id", "tpl-1"); err != nil {
		t.Fatal(err)
	}
}

func TestCreateRequiresNameOrTemplate(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })

	err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "create")
	if err == nil || !strings.Contains(err.Error(), "--name is required unless --template-id") {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateScheduleBody(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/workspaces/42/dataflows/df-1" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := map[string]interface{}{
			"name": "Weekly",
			"schedule_config": map[string]interface{}{
				"interval":     "hourly",
				"week_days":    []interface{}{float64(0), float64(1)},
				"active_hours": map[string]interface{}{"start": float64(9), "end": float64(18)},
				"timezone":     "Europe/Berlin",
				"enabled":      true,
			},
		}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v\nwant %v", body, want)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{})
	})

	err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "update", "df-1",
		"--name", "Weekly", "--interval", "hourly", "--week-day", "0", "--week-day", "1",
		"--active-hours-start", "9", "--active-hours-end", "18", "--timezone", "Europe/Berlin", "--schedule-enabled")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "Dataflow df-1 updated.") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestUpdateSendsOnlyEnabledToggle(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		want := map[string]interface{}{"schedule_config": map[string]interface{}{"enabled": false}}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "update", "df-1", "--schedule-enabled=false"); err != nil {
		t.Fatal(err)
	}
}

// The CLI only checks how a command is used. Whether an interval, day or time
// is valid is the API's call (see README, "Validation").
func TestUpdateUsageErrors(t *testing.T) {
	tests := map[string]struct {
		args []string
		want string
	}{
		"nothing":        {[]string{"update", "df-1"}, "nothing to update"},
		"half hours set": {[]string{"update", "df-1", "--active-hours-start", "9"}, "must be set together"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
			err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want error containing %q", err, tt.want)
			}
			if code := cmdutil.ExitCode(err); code != 2 {
				t.Errorf("exit code = %d, want 2 (usage error)", code)
			}
		})
	}
}

func TestUpdateLeavesValueValidationToTheAPI(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		want := map[string]interface{}{"schedule_config": map[string]interface{}{
			"interval":  "weekly",
			"week_days": []interface{}{float64(7)},
			"month_day": float64(31),
		}}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("values must be forwarded untouched, body = %v", body)
		}
		cmdtest.JSON(w, 422, map[string]interface{}{"error": map[string]string{"message": "Unknown interval"}})
	})

	err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "update", "df-1",
		"--interval", "weekly", "--week-day", "7", "--month-day", "31")
	if err == nil || !strings.Contains(err.Error(), "Unknown interval") {
		t.Fatalf("expected the API message, got %v", err)
	}
	if code := cmdutil.ExitCode(err); code != 1 {
		t.Errorf("exit code = %d, want 1 (runtime error)", code)
	}
}

func TestRun(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/workspaces/42/dataflows/df-1/executions" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if body := cmdtest.DecodeBody(t, r); body["run_mode"] != "preview" {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "run", "df-1", "--mode", "preview"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "queued (preview run)") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestRunDefaultsToFull(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if body := cmdtest.DecodeBody(t, r); body["run_mode"] != "full" {
			t.Errorf("body = %v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "run", "df-1"); err != nil {
		t.Fatal(err)
	}
}

func TestRunUnprocessableSurfacesAPIMessage(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 422, map[string]interface{}{"error": map[string]string{"message": "Dataflow is not ready"}})
	})

	err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "run", "df-1")
	if err == nil || !strings.Contains(err.Error(), "422") || !strings.Contains(err.Error(), "Dataflow is not ready") {
		t.Fatalf("got %v", err)
	}
}

func TestSetupAlwaysPrintsJSON(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, map[string]interface{}{"id": "df-1", "sources": []interface{}{}})
	})

	if err := cmdtest.Run(dataflows.NewCmdDataflows(env.Factory), "setup", "df-1"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(env.Out.String(), "{\n  \"id\": \"df-1\"") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}
