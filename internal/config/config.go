// Package config resolves settings from flags, COUPLER_* env vars and
// ~/.coupler.yaml (in that order of precedence).
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	KeyAPIToken    = "api-token"
	KeyWorkspaceID = "workspace-id"
	KeyAPIURL      = "api-url"
	KeyOutput      = "output"
)

// Load binds COUPLER_* env vars to v and reads ~/.coupler.yaml into it.
//
// Only the config file in the user's home directory is read. The current
// directory is deliberately not searched: a .coupler.yaml inside an untrusted
// checkout could otherwise point api-url at a hostile host and receive the
// API token.
//
// A missing file is not an error. A file that exists but cannot be read or
// parsed is, so a typo does not silently drop the saved token.
func Load(v *viper.Viper) error {
	v.SetEnvPrefix("COUPLER")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	path, err := ConfigFilePath()
	if err != nil {
		// No home directory: fall back to flags and environment only.
		return nil
	}
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read config file %s: %w", path, err)
	}
	return nil
}

func RequireAPIToken(v *viper.Viper) (string, error) {
	token := v.GetString(KeyAPIToken)
	if token == "" {
		return "", errors.New("api-token is required: set via --api-token, --api-token-stdin, COUPLER_API_TOKEN, or run `coupler configure`")
	}
	return token, nil
}

func RequireWorkspaceID(v *viper.Viper) (string, error) {
	id := v.GetString(KeyWorkspaceID)
	if id == "" {
		return "", errors.New("workspace-id is required: set via --workspace-id, COUPLER_WORKSPACE_ID, or run `coupler configure` (list ids with `coupler workspaces list`)")
	}
	return id, nil
}

// ConfigFilePath returns ~/.coupler.yaml. It fails when the home directory is
// unknown rather than falling back to a path relative to the current directory.
func ConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".coupler.yaml"), nil
}
