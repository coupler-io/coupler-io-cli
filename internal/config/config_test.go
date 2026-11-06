package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/coupler-io/coupler-io-cli/internal/config"
)

// useHome points the home directory at a temp dir and returns it.
func useHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	useHome(t)
	v := viper.New()
	if err := config.Load(v); err != nil {
		t.Fatalf("missing config file should be fine, got %v", err)
	}
}

func TestLoadReadsHomeFileAndEnvWins(t *testing.T) {
	home := useHome(t)
	writeFile(t, filepath.Join(home, ".coupler.yaml"), "api-token: from-file\nworkspace-id: \"7\"\n")
	t.Setenv("COUPLER_WORKSPACE_ID", "99")

	v := viper.New()
	if err := config.Load(v); err != nil {
		t.Fatal(err)
	}
	if got := v.GetString(config.KeyAPIToken); got != "from-file" {
		t.Errorf("api-token = %q", got)
	}
	if got := v.GetString(config.KeyWorkspaceID); got != "99" {
		t.Errorf("env should override file, workspace-id = %q", got)
	}
}

func TestLoadReportsMalformedFile(t *testing.T) {
	home := useHome(t)
	path := filepath.Join(home, ".coupler.yaml")
	writeFile(t, path, "api-token: [unterminated\n")

	err := config.Load(viper.New())
	if err == nil {
		t.Fatal("expected an error for a malformed config file")
	}
	if got := err.Error(); !strings.Contains(got, path) {
		t.Errorf("error should name the file %s, got %q", path, got)
	}
}

func TestLoadIgnoresConfigInWorkingDirectory(t *testing.T) {
	useHome(t)
	cwd := t.TempDir()
	writeFile(t, filepath.Join(cwd, ".coupler.yaml"), "api-token: stolen\napi-url: https://evil.example\n")
	t.Chdir(cwd)

	v := viper.New()
	if err := config.Load(v); err != nil {
		t.Fatal(err)
	}
	if v.GetString(config.KeyAPIToken) != "" || v.GetString(config.KeyAPIURL) != "" {
		t.Errorf("config in the working directory must not be read: token=%q url=%q",
			v.GetString(config.KeyAPIToken), v.GetString(config.KeyAPIURL))
	}
}

func TestRequireErrors(t *testing.T) {
	v := viper.New()
	if _, err := config.RequireAPIToken(v); err == nil {
		t.Error("expected error for missing token")
	}
	if _, err := config.RequireWorkspaceID(v); err == nil {
		t.Error("expected error for missing workspace")
	}
	v.Set(config.KeyAPIToken, "t")
	v.Set(config.KeyWorkspaceID, "1")
	if tok, err := config.RequireAPIToken(v); err != nil || tok != "t" {
		t.Errorf("got %q, %v", tok, err)
	}
	if id, err := config.RequireWorkspaceID(v); err != nil || id != "1" {
		t.Errorf("got %q, %v", id, err)
	}
}
