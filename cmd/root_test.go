package cmd_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/coupler-io/coupler-io-cli/cmd"
	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

// run builds the full command tree, so flags are parsed and bound like in the
// real binary. cmdtest.Setup presets viper; it is replaced so flags decide.
func run(t *testing.T, env *cmdtest.Env, args ...string) error {
	t.Helper()
	env.Factory.Viper = viper.New()
	root := cmd.NewRootCmd(env.Factory)
	root.SetOut(env.Out)
	return cmdtest.Run(root, args...)
}

func TestWorkspacesList(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"workspaces": []map[string]interface{}{
			{"id": 42, "name": "Marketing", "organization": "Acme"},
		}})
	})

	if err := run(t, env, "workspaces", "list"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ID", "42", "Marketing", "Acme"} {
		if !strings.Contains(env.Out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, env.Out.String())
		}
	}
}

func TestOutputFlagJSONAndInvalid(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, map[string]interface{}{"workspaces": []map[string]interface{}{{"id": 1, "name": "A", "organization": "O"}}})
	})

	if err := run(t, env, "workspaces", "list", "-o", "json"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), `"workspaces": [`) {
		t.Errorf("json output should be the full body:\n%s", env.Out.String())
	}

	err := run(t, env, "workspaces", "list", "-o", "yaml")
	if err == nil || !strings.Contains(err.Error(), `unknown output format "yaml"`) {
		t.Fatalf("got %v", err)
	}
}

func TestWorkspaceFlagAndEnv(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces/7/dataflows" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		cmdtest.JSON(w, 200, []interface{}{})
	})
	if err := run(t, env, "dataflows", "list", "-w", "7"); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialsListQuery(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces/42/credentials" || r.URL.Query().Get("type") != "source" || r.URL.Query().Get("provider") != "google" {
			t.Errorf("unexpected request %s", r.URL)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{
			"credentials": []map[string]interface{}{{"id": 15, "type": "source", "provider": "google", "name": "me@acme.com", "status": "active"}},
			"setup_url":   "https://app.coupler.io/app/credentials",
		})
	})

	if err := run(t, env, "credentials", "list", "-w", "42", "--type", "source", "--provider", "google"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"15", "me@acme.com", "Add credentials at: https://app.coupler.io/app/credentials"} {
		if !strings.Contains(env.Out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, env.Out.String())
		}
	}

	err := run(t, env, "credentials", "list", "-w", "42", "--type", "bogus")
	if err == nil || !strings.Contains(err.Error(), "--type must be source or destination") {
		t.Fatalf("got %v", err)
	}
}

func TestIntegrationsGetShowsParams(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/integrations/sources/csv" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{
			"key": "csv", "name": "CSV", "credential_provider": nil,
			"params": []map[string]interface{}{{"name": "url", "title": "File URL", "type": "string", "required": true, "helper_text": "Public link"}},
		})
	})

	if err := run(t, env, "integrations", "sources", "get", "csv"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"CSV", "Params:", "url", "File URL", "true", "Public link"} {
		if !strings.Contains(env.Out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, env.Out.String())
		}
	}
}

func TestIntegrationsDestinationsList(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/integrations/destinations" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"integrations": []map[string]string{{"key": "google_spreadsheet", "name": "Google Sheets"}}})
	})

	if err := run(t, env, "integrations", "destinations", "list"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "google_spreadsheet") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestFieldOptionsQuery(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces/42/integrations/sources/gsheet/field_options" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("field") != "sheetsList" || q.Get("credential_id") != "15" || q.Get("params[spreadsheetURL]") != "https://x" {
			t.Errorf("unexpected query %v", q)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"field": "sheetsList", "options": []string{"Sheet1"}})
	})

	err := run(t, env, "integrations", "sources", "field-options", "gsheet", "-w", "42",
		"--field", "sheetsList", "--credential-id", "15", "--param", "spreadsheetURL=https://x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "Sheet1") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestTemplatesList(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, []map[string]interface{}{
			{"id": "t-1", "name": "Ads dashboard", "type": "DashboardTemplate", "sources": []string{"facebook-ads"}, "categories": []string{"Marketing"}},
		})
	})

	if err := run(t, env, "templates", "list"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"t-1", "Ads dashboard", "facebook-ads", "Marketing"} {
		if !strings.Contains(env.Out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, env.Out.String())
		}
	}
}

func TestSkills(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/workspaces/42/agent_skills":
			if r.URL.Query().Get("query") != "ads" {
				t.Errorf("unexpected query %v", r.URL.Query())
			}
			cmdtest.JSON(w, 200, map[string]interface{}{"skills": []map[string]interface{}{{"slug": "ads-report", "name": "Ads report"}}})
		case "/v1/workspaces/42/agent_skills/ads-report/prompt":
			cmdtest.JSON(w, 200, map[string]string{"content": "Step 1: do the thing"})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})

	if err := run(t, env, "skills", "list", "-w", "42", "--query", "ads"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "ads-report") {
		t.Errorf("unexpected list output:\n%s", env.Out.String())
	}

	env.Out.Reset()
	if err := run(t, env, "skills", "prompt", "ads-report", "-w", "42"); err != nil {
		t.Fatal(err)
	}
	if env.Out.String() != "Step 1: do the thing\n" {
		t.Errorf("prompt output = %q", env.Out.String())
	}
}

func TestMissingTokenWithoutOverride(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
	env.Factory.ClientOverride = nil
	t.Setenv("COUPLER_API_TOKEN", "")

	err := run(t, env, "workspaces", "list")
	if err == nil || !strings.Contains(err.Error(), "api-token is required") {
		t.Fatalf("got %v", err)
	}
}

func TestUsageErrorsExitWithCode2(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"workspaces", "list", "--bogus"}},
		{"unknown root command", []string{"bogus"}},
		{"unknown group subcommand", []string{"dataflows", "bogus"}},
		{"wrong argument count", []string{"dataflows", "get"}},
		{"invalid output format", []string{"workspaces", "list", "-o", "yaml"}},
		{"missing required flag", []string{"datasets", "create", "-w", "42"}},
		{"invalid completion shell", []string{"completion", "tcsh"}},
		{"malformed --param", []string{"integrations", "sources", "field-options", "gsheet", "-w", "42", "--field", "f", "--param", "novalue"}},
	}
	for _, tc := range cases {
		env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
			cmdtest.JSON(w, 200, map[string]interface{}{})
		})
		err := run(t, env, tc.args...)
		if err == nil {
			t.Errorf("%s: expected an error", tc.name)
			continue
		}
		if got := cmdutil.ExitCode(err); got != cmdutil.ExitUsage {
			t.Errorf("%s: exit code = %d (%v), want %d", tc.name, got, err, cmdutil.ExitUsage)
		}
	}
}

func TestRuntimeErrorsExitWithCode1(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 500, map[string]string{"message": "boom"})
	})
	err := run(t, env, "workspaces", "list")
	if err == nil || cmdutil.ExitCode(err) != cmdutil.ExitError {
		t.Fatalf("got %v (code %d)", err, cmdutil.ExitCode(err))
	}
}

func TestGroupWithoutSubcommandPrintsHelp(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
	root := cmd.NewRootCmd(env.Factory)
	root.SetOut(env.Out)
	if err := cmdtest.Run(root, "dataflows"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "Usage:") {
		t.Errorf("expected help output, got:\n%s", env.Out.String())
	}
}

func TestAPITokenStdin(t *testing.T) {
	var gotAuth string
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		cmdtest.JSON(w, 200, map[string]interface{}{"workspaces": []interface{}{}})
	})
	env.Factory.ClientOverride = nil
	env.Factory.Viper = viper.New()
	env.Factory.IOStreams.In = strings.NewReader("  secret-token\n")

	root := cmd.NewRootCmd(env.Factory)
	if err := cmdtest.Run(root, "workspaces", "list", "--api-token-stdin", "--api-url", env.URL); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
}

func TestAPITokenStdinErrors(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })

	env.Factory.IOStreams.In = strings.NewReader("")
	err := run(t, env, "workspaces", "list", "--api-token-stdin")
	if err == nil || cmdutil.ExitCode(err) != cmdutil.ExitUsage || !strings.Contains(err.Error(), "empty") {
		t.Errorf("empty stdin: got %v", err)
	}

	env.Factory.IOStreams.In = strings.NewReader("tok")
	err = run(t, env, "workspaces", "list", "--api-token-stdin", "--api-token", "x")
	if err == nil || cmdutil.ExitCode(err) != cmdutil.ExitUsage || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("both flags: got %v", err)
	}
}

func TestBadConfigFileBlocksCommandsButNotConfigure(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, map[string]interface{}{"workspaces": []interface{}{}})
	})
	env.Factory.ConfigErr = errors.New("read config file ~/.coupler.yaml: bad yaml")

	err := run(t, env, "workspaces", "list")
	if err == nil || !strings.Contains(err.Error(), "bad yaml") {
		t.Fatalf("expected the config error, got %v", err)
	}

	// configure must get past the config check so it can overwrite the file.
	// It fails later (no token), not with the config error.
	err = run(t, env, "configure")
	if err == nil || strings.Contains(err.Error(), "bad yaml") {
		t.Fatalf("configure should not report the config error, got %v", err)
	}

	env.Out.Reset()
	if err := run(t, env, "completion", "bash"); err != nil {
		t.Fatalf("completion should ignore the config error, got %v", err)
	}
}
