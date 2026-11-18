package cmdutil

import (
	"net/url"
	"strings"

	"github.com/coupler-io/coupler-io-cli/internal/config"
)

// Path builds an API path such as /v1/integrations/sources/csv, escaping every segment.
func Path(segments ...string) string {
	parts := make([]string, 0, len(segments)+1)
	parts = append(parts, "/v1")
	for _, s := range segments {
		parts = append(parts, url.PathEscape(s))
	}
	return strings.Join(parts, "/")
}

// WorkspacePath builds /v1/workspaces/<workspace-id>/<segments...> and fails
// when no workspace is configured.
func (f *Factory) WorkspacePath(segments ...string) (string, error) {
	id, err := config.RequireWorkspaceID(f.Viper)
	if err != nil {
		return "", err
	}
	return Path(append([]string{"workspaces", id}, segments...)...), nil
}
