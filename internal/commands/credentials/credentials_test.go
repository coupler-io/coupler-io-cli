package credentials_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/commands/credentials"
)

func credentialsResponse() map[string]interface{} {
	return map[string]interface{}{
		"credentials": []map[string]interface{}{
			{"id": 15, "type": "source", "provider": "google", "name": "me@acme.com", "status": "active"},
		},
		"setup_url": "https://app.coupler.io/app/credentials",
	}
}

func TestListSendsFilters(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/v1/workspaces/42/credentials" || q.Get("type") != "source" || q.Get("provider") != "google" {
			t.Errorf("unexpected request %s", r.URL)
		}
		cmdtest.JSON(w, 200, credentialsResponse())
	})

	if err := cmdtest.Run(credentials.NewCmdCredentials(env.Factory), "list", "--type", "source", "--provider", "google"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"15", "me@acme.com", "Add credentials at: https://app.coupler.io/app/credentials"} {
		if !strings.Contains(env.Out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, env.Out.String())
		}
	}
}

func TestListOmitsUnsetFilters(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("no filters were given, query = %q", r.URL.RawQuery)
		}
		cmdtest.JSON(w, 200, credentialsResponse())
	})

	if err := cmdtest.Run(credentials.NewCmdCredentials(env.Factory), "list"); err != nil {
		t.Fatal(err)
	}
}

// Whether a type or provider is known is the API's call, so the CLI forwards
// the value and reports the API's answer as a runtime error.
func TestListForwardsFilterValuesToTheAPI(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "bogus" {
			t.Errorf("type must be forwarded untouched, query = %q", r.URL.RawQuery)
		}
		cmdtest.JSON(w, 422, map[string]interface{}{"error": map[string]string{"message": "type must be one of: source, destination"}})
	})

	err := cmdtest.Run(credentials.NewCmdCredentials(env.Factory), "list", "--type", "bogus")
	if err == nil || !strings.Contains(err.Error(), "type must be one of") {
		t.Fatalf("expected the API message, got %v", err)
	}
	if code := cmdutil.ExitCode(err); code != cmdutil.ExitError {
		t.Errorf("exit code = %d, want %d", code, cmdutil.ExitError)
	}
}

func TestListJSONIsTheRawBody(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, credentialsResponse())
	})
	env.Factory.Viper.Set("output", "json")

	if err := cmdtest.Run(credentials.NewCmdCredentials(env.Factory), "list"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), `"setup_url"`) {
		t.Errorf("json output should keep fields the table hides:\n%s", env.Out.String())
	}
}
