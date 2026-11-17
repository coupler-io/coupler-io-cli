package cmd_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

// Standard input can be read once. --api-token-stdin and a "-" value must not
// silently compete for it: the second reader would get an empty string and
// fail with a confusing message.
func TestAPITokenStdinCannotBeCombinedWithDashValue(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
	env.Factory.ClientOverride = nil
	env.Factory.IOStreams.In = strings.NewReader("secret-token\n")

	err := run(t, env, "datasets", "query", "ds-1", "-s", "-", "--api-token-stdin", "--api-url", env.URL)
	if err == nil || cmdutil.ExitCode(err) != cmdutil.ExitUsage {
		t.Fatalf("expected a usage error, got %v", err)
	}
	for _, want := range []string{"standard input is already used by --api-token-stdin", "@file"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

func TestTwoDashValuesAreAUsageError(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
	env.Factory.IOStreams.In = strings.NewReader("select 1\n")

	err := run(t, env, "datasets", "update", "ds-1", "--ai-context", "-", "--query", "-")
	if err == nil || cmdutil.ExitCode(err) != cmdutil.ExitUsage || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("expected a usage error, got %v", err)
	}
}
