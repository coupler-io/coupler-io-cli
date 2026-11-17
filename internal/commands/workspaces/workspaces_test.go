package workspaces_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/commands/workspaces"
)

func TestList(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/workspaces" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"workspaces": []map[string]interface{}{
			{"id": 42, "name": "Marketing", "organization": "Acme"},
		}})
	})

	if err := cmdtest.Run(workspaces.NewCmdWorkspaces(env.Factory), "list"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ID", "42", "Marketing", "Acme"} {
		if !strings.Contains(env.Out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, env.Out.String())
		}
	}
}

func TestListJSONIsTheRawBody(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, map[string]interface{}{"workspaces": []map[string]interface{}{
			{"id": 1, "name": "A", "organization": "O", "plan": "free"},
		}})
	})
	env.Factory.Viper.Set("output", "json")

	if err := cmdtest.Run(workspaces.NewCmdWorkspaces(env.Factory), "list"); err != nil {
		t.Fatal(err)
	}
	// Fields the table hides must survive in JSON output.
	if !strings.Contains(env.Out.String(), `"plan": "free"`) {
		t.Errorf("json output should be the untouched API body:\n%s", env.Out.String())
	}
}

func TestListAPIError(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 401, map[string]interface{}{"error": map[string]string{"message": "Invalid token"}})
	})

	err := cmdtest.Run(workspaces.NewCmdWorkspaces(env.Factory), "list")
	if err == nil || !strings.Contains(err.Error(), "Invalid token") {
		t.Fatalf("expected the API message, got %v", err)
	}
}

func TestListRejectsArguments(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })

	if err := cmdtest.Run(workspaces.NewCmdWorkspaces(env.Factory), "list", "extra"); err == nil {
		t.Fatal("expected an error for a positional argument")
	}
}
