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

func TestMissingTokenWithoutOverride(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
	env.Factory.ClientOverride = nil
	t.Setenv("COUPLER_API_TOKEN", "")

	err := run(t, env, "workspaces", "list")
	if err == nil || !strings.Contains(err.Error(), "api-token is required") {
		t.Fatalf("got %v", err)
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
