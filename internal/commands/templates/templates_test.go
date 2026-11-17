package templates_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/commands/templates"
)

func templatesResponse() []map[string]interface{} {
	return []map[string]interface{}{
		{"id": "t-1", "name": "Ads dashboard", "type": "DashboardTemplate", "sources": []string{"facebook-ads"}, "categories": []string{"Marketing"}},
	}
}

func TestList(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/templates" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		cmdtest.JSON(w, 200, templatesResponse())
	})

	if err := cmdtest.Run(templates.NewCmdTemplates(env.Factory), "list"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"t-1", "Ads dashboard", "facebook-ads", "Marketing"} {
		if !strings.Contains(env.Out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, env.Out.String())
		}
	}
}

func TestListJSONIsTheRawBody(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, templatesResponse())
	})
	env.Factory.Viper.Set("output", "json")

	if err := cmdtest.Run(templates.NewCmdTemplates(env.Factory), "list"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), `"type": "DashboardTemplate"`) {
		t.Errorf("json output should keep fields the table hides:\n%s", env.Out.String())
	}
}
