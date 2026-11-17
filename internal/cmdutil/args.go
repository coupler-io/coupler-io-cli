package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// AnnotationIgnoreConfigError, set (to any value) on a command, lets it run
// even when ~/.coupler-io.yaml is unreadable, e.g. so `configure` can repair it.
const AnnotationIgnoreConfigError = "coupler.io/ignore-config-error"

// RequireFlag returns a usage error when a required string flag is empty.
func RequireFlag(name, value string) error {
	if value == "" {
		return UsageErrorf("--%s is required", name)
	}
	return nil
}

// ReadArg resolves a flag value that may point elsewhere: "@path" reads a
// file and "-" reads stdin, which can be read only once per run (see
// Factory.StdinUsedBy). Any other value is returned unchanged.
func (f *Factory) ReadArg(value string) (string, error) {
	switch {
	case value == "-":
		if f.StdinUsedBy != "" {
			return "", UsageErrorf("standard input is already used by %s; pass this value as @file instead", f.StdinUsedBy)
		}
		f.StdinUsedBy = "another flag"
		data, err := io.ReadAll(f.IOStreams.In)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	case strings.HasPrefix(value, "@"):
		data, err := os.ReadFile(strings.TrimPrefix(value, "@"))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", strings.TrimPrefix(value, "@"), err)
		}
		return strings.TrimSpace(string(data)), nil
	default:
		return value, nil
	}
}

// ParseParams builds an integration params object from --params (a JSON
// object, "@file" or "-") and repeated --param key=value flags. --param
// values are always strings and override the same key from --params. It
// returns nil when neither is given.
func (f *Factory) ParseParams(jsonArg string, pairs []string) (map[string]interface{}, error) {
	if jsonArg == "" && len(pairs) == 0 {
		return nil, nil
	}

	params := map[string]interface{}{}
	if jsonArg != "" {
		raw, err := f.ReadArg(jsonArg)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &params); err != nil {
			return nil, UsageErrorf("--params must be a JSON object: %w", err)
		}
	}

	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return nil, UsageErrorf("--param %q must be in key=value form", pair)
		}
		params[key] = value
	}
	return params, nil
}
