package skills_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/commands/skills"
)

func TestListSendsQuery(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces/42/agent_skills" || r.URL.Query().Get("query") != "ads" {
			t.Errorf("unexpected request %s", r.URL)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"skills": []map[string]interface{}{{"slug": "ads-report", "name": "Ads report"}}})
	})

	if err := cmdtest.Run(skills.NewCmdSkills(env.Factory), "list", "--query", "ads"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"SLUG", "ads-report", "Ads report"} {
		if !strings.Contains(env.Out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, env.Out.String())
		}
	}
}

func TestListOmitsEmptyQuery(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"skills": []interface{}{}})
	})

	if err := cmdtest.Run(skills.NewCmdSkills(env.Factory), "list"); err != nil {
		t.Fatal(err)
	}
}

func TestPromptPrintsContent(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces/42/agent_skills/ads-report/prompt" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		cmdtest.JSON(w, 200, map[string]string{"content": "Step 1: do the thing"})
	})

	if err := cmdtest.Run(skills.NewCmdSkills(env.Factory), "prompt", "ads-report"); err != nil {
		t.Fatal(err)
	}
	if env.Out.String() != "Step 1: do the thing\n" {
		t.Errorf("prompt output = %q", env.Out.String())
	}
}

func TestPromptJSONIsTheRawBody(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, map[string]string{"content": "Step 1", "slug": "ads-report"})
	})
	env.Factory.Viper.Set("output", "json")

	if err := cmdtest.Run(skills.NewCmdSkills(env.Factory), "prompt", "ads-report"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), `"slug": "ads-report"`) {
		t.Errorf("json output should be the untouched API body:\n%s", env.Out.String())
	}
}

func TestPromptEscapesTheSlug(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); got != "/v1/workspaces/42/agent_skills/a%2Fb/prompt" {
			t.Errorf("escaped path = %s", got)
		}
		cmdtest.JSON(w, 200, map[string]string{"content": "x"})
	})

	if err := cmdtest.Run(skills.NewCmdSkills(env.Factory), "prompt", "a/b"); err != nil {
		t.Fatal(err)
	}
}

func TestPromptRequiresSlug(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })

	if err := cmdtest.Run(skills.NewCmdSkills(env.Factory), "prompt"); err == nil {
		t.Fatal("expected an error without a slug")
	}
}
