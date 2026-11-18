package configure_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/commands/configure"
)

func workspacesHandler(t *testing.T, workspaces ...map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header %q", r.Header.Get("Authorization"))
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"workspaces": workspaces})
	}
}

// useTempHome points ~/.coupler.yaml at a temp dir and returns its path.
func useTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return filepath.Join(home, ".coupler.yaml")
}

func TestConfigureSingleWorkspace(t *testing.T) {
	path := useTempHome(t)
	env := cmdtest.Setup(t, workspacesHandler(t, map[string]interface{}{"id": 42, "name": "Marketing", "organization": "Acme"}))
	env.Factory.Viper.Set("api-token", "test-token")
	env.Factory.Viper.Set("workspace-id", "")

	if err := cmdtest.Run(configure.NewCmdConfigure(env.Factory)); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]string
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["api-token"] != "test-token" || saved["workspace-id"] != "42" {
		t.Errorf("unexpected config file:\n%s", data)
	}
	if strings.Contains(string(data), "api-url") {
		t.Errorf("default api url should not be saved:\n%s", data)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("config file mode = %v, want 0600", info.Mode().Perm())
	}
	if !strings.Contains(env.Out.String(), "Marketing") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestConfigureTightensExistingFilePermissions(t *testing.T) {
	path := useTempHome(t)
	if err := os.WriteFile(path, []byte("old: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := cmdtest.Setup(t, workspacesHandler(t, map[string]interface{}{"id": 1, "name": "A", "organization": "O"}))
	env.Factory.Viper.Set("api-token", "test-token")
	env.Factory.Viper.Set("workspace-id", "")

	if err := cmdtest.Run(configure.NewCmdConfigure(env.Factory)); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("config file mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestConfigureExplicitWorkspace(t *testing.T) {
	path := useTempHome(t)
	env := cmdtest.Setup(t, workspacesHandler(t,
		map[string]interface{}{"id": 1, "name": "One", "organization": "Acme"},
		map[string]interface{}{"id": 2, "name": "Two", "organization": "Acme"}))
	env.Factory.Viper.Set("api-token", "test-token")
	env.Factory.Viper.Set("workspace-id", "2")

	if err := cmdtest.Run(configure.NewCmdConfigure(env.Factory)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "2") || !strings.Contains(env.Out.String(), "Two") {
		t.Errorf("config:\n%s\noutput:\n%s", data, env.Out.String())
	}
}

func TestConfigureUnknownWorkspace(t *testing.T) {
	useTempHome(t)
	env := cmdtest.Setup(t, workspacesHandler(t, map[string]interface{}{"id": 1, "name": "One", "organization": "Acme"}))
	env.Factory.Viper.Set("api-token", "test-token")
	env.Factory.Viper.Set("workspace-id", "99")

	err := cmdtest.Run(configure.NewCmdConfigure(env.Factory))
	if err == nil || !strings.Contains(err.Error(), "workspace 99 is not accessible") {
		t.Fatalf("got %v", err)
	}
}

func TestConfigurePromptsWhenSeveralWorkspaces(t *testing.T) {
	path := useTempHome(t)
	env := cmdtest.Setup(t, workspacesHandler(t,
		map[string]interface{}{"id": 10, "name": "One", "organization": "Acme"},
		map[string]interface{}{"id": 20, "name": "Two", "organization": "Acme"}))
	env.Factory.Viper.Set("api-token", "test-token")
	env.Factory.Viper.Set("workspace-id", "")
	env.Factory.IOStreams.In = strings.NewReader("2\n")

	if err := cmdtest.Run(configure.NewCmdConfigure(env.Factory)); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "20") {
		t.Errorf("expected workspace 20 to be saved:\n%s", data)
	}
	if !strings.Contains(env.Out.String(), "Select a workspace [1-2]") {
		t.Errorf("expected prompt in output:\n%s", env.Out.String())
	}
}

func TestConfigureSeveralWorkspacesWithoutInput(t *testing.T) {
	path := useTempHome(t)
	env := cmdtest.Setup(t, workspacesHandler(t,
		map[string]interface{}{"id": 10, "name": "One", "organization": "Acme"},
		map[string]interface{}{"id": 20, "name": "Two", "organization": "Acme"}))
	env.Factory.Viper.Set("api-token", "test-token")
	env.Factory.Viper.Set("workspace-id", "")

	err := cmdtest.Run(configure.NewCmdConfigure(env.Factory))
	if err == nil || !strings.Contains(err.Error(), "pass --workspace-id") {
		t.Fatalf("got %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Error("config must not be written on failure")
	}
}

func TestConfigureInvalidToken(t *testing.T) {
	path := useTempHome(t)
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 401, map[string]interface{}{"error": map[string]string{"message": "Invalid token"}})
	})
	env.Factory.Viper.Set("api-token", "bad")

	err := cmdtest.Run(configure.NewCmdConfigure(env.Factory))
	if err == nil || !strings.Contains(err.Error(), "could not validate the API token") || !strings.Contains(err.Error(), "Invalid token") {
		t.Fatalf("got %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Error("config must not be written on failure")
	}
}

func TestConfigureRequiresToken(t *testing.T) {
	useTempHome(t)
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })

	err := cmdtest.Run(configure.NewCmdConfigure(env.Factory))
	if err == nil || !strings.Contains(err.Error(), "api-token is required") {
		t.Fatalf("got %v", err)
	}
}
