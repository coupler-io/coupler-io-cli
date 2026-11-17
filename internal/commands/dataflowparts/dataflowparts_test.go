package dataflowparts_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/commands/dataflowparts"
)

// Behaviour shared by sources and destinations is tested once, for both specs.
// Kind-specific behaviour (paths, the enabled flag) lives next to each kind.
var specs = map[string]dataflowparts.Spec{
	"source":      {Noun: "source", Plural: "sources", IntegrationExample: "csv", SupportsEnabled: true},
	"destination": {Noun: "destination", Plural: "destinations", IntegrationExample: "gsheet"},
}

func newCmd(env *cmdtest.Env, kind string) *cobra.Command {
	return dataflowparts.NewCmd(env.Factory, specs[kind])
}

func eachKind(t *testing.T, fn func(t *testing.T, kind string, spec dataflowparts.Spec)) {
	for kind, spec := range specs {
		t.Run(kind, func(t *testing.T) { fn(t, kind, spec) })
	}
}

func TestCreateOmitsUnsetOptionalFields(t *testing.T) {
	eachKind(t, func(t *testing.T, kind string, spec dataflowparts.Spec) {
		env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/workspaces/42/dataflows/df-1/"+spec.Plural {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, map[string]interface{}{"integration_key": "x"}) {
				t.Errorf("body = %v", body)
			}
			cmdtest.JSON(w, 201, map[string]interface{}{"id": "1"})
		})

		if err := cmdtest.Run(newCmd(env, kind), "create", "--dataflow-id", "df-1", "--integration", "x"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(env.Out.String(), "ID") {
			t.Errorf("unexpected output:\n%s", env.Out.String())
		}
	})
}

func TestParamFlagWinsOverParamsJSON(t *testing.T) {
	eachKind(t, func(t *testing.T, kind string, _ dataflowparts.Spec) {
		env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
			want := map[string]interface{}{"a": "from-flag", "b": float64(2)}
			if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body["params"], want) {
				t.Errorf("params = %v, want %v", body["params"], want)
			}
			cmdtest.JSON(w, 201, map[string]interface{}{"id": "1"})
		})

		err := cmdtest.Run(newCmd(env, kind), "create", "--dataflow-id", "df-1", "--integration", "x",
			"--params", `{"a":"from-json","b":2}`, "--param", "a=from-flag")
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestUsageErrors(t *testing.T) {
	tests := map[string]struct {
		args []string
		want string
	}{
		"create without dataflow":    {[]string{"create", "--integration", "x"}, "--dataflow-id is required"},
		"create without integration": {[]string{"create", "--dataflow-id", "df-1"}, "--integration is required"},
		"create with bad param":      {[]string{"create", "--dataflow-id", "df-1", "--integration", "x", "--param", "novalue"}, "key=value"},
		"update with nothing":        {[]string{"update", "1"}, "nothing to update"},
		"update with bad params":     {[]string{"update", "1", "--params", "[1]"}, "JSON object"},
	}
	eachKind(t, func(t *testing.T, kind string, _ dataflowparts.Spec) {
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
				err := cmdtest.Run(newCmd(env, kind), tt.args...)
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("got %v, want error containing %q", err, tt.want)
				}
				if code := cmdutil.ExitCode(err); code != cmdutil.ExitUsage {
					t.Errorf("exit code = %d, want %d", code, cmdutil.ExitUsage)
				}
			})
		}
	})
}

// Which lifecycle states exist, and whether pausing is allowed on the plan, is
// the API's call. The CLI forwards the value and reports the API's answer.
func TestLifecycleStateIsValidatedByTheAPI(t *testing.T) {
	eachKind(t, func(t *testing.T, kind string, _ dataflowparts.Spec) {
		env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
			if body := cmdtest.DecodeBody(t, r); body["lifecycle_state"] != "asleep" {
				t.Errorf("value must be forwarded untouched, body = %v", body)
			}
			cmdtest.JSON(w, 422, map[string]interface{}{"error": map[string]string{"message": "lifecycle_state must be one of: active, paused, disabled"}})
		})

		err := cmdtest.Run(newCmd(env, kind), "update", "1", "--lifecycle-state", "asleep")
		if err == nil || !strings.Contains(err.Error(), "lifecycle_state must be one of") {
			t.Fatalf("expected the API message, got %v", err)
		}
		if code := cmdutil.ExitCode(err); code != cmdutil.ExitError {
			t.Errorf("exit code = %d, want %d (runtime error)", code, cmdutil.ExitError)
		}
	})
}

func TestEnabledFlagOnlyWhenSupported(t *testing.T) {
	eachKind(t, func(t *testing.T, kind string, spec dataflowparts.Spec) {
		env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
			if !spec.SupportsEnabled {
				t.Error("no request expected")
			}
			cmdtest.JSON(w, 200, map[string]interface{}{"id": "1"})
		})

		err := cmdtest.Run(newCmd(env, kind), "update", "1", "--enabled=false")
		if spec.SupportsEnabled && err != nil {
			t.Fatal(err)
		}
		if !spec.SupportsEnabled && (err == nil || !strings.Contains(err.Error(), "unknown flag: --enabled")) {
			t.Fatalf("got %v", err)
		}
	})
}
