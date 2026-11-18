package cmdutil_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func TestExitCode(t *testing.T) {
	plain := errors.New("boom")
	usage := cmdutil.UsageErrorf("--x is required")

	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, cmdutil.ExitOK},
		{"runtime error", plain, cmdutil.ExitError},
		{"usage error", usage, cmdutil.ExitUsage},
		{"wrapped usage error", fmt.Errorf("ctx: %w", usage), cmdutil.ExitUsage},
		{"wrapped plain error", cmdutil.WrapUsage(plain), cmdutil.ExitUsage},
	}
	for _, tc := range cases {
		if got := cmdutil.ExitCode(tc.err); got != tc.want {
			t.Errorf("%s: ExitCode = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestWrapUsageKeepsMessageAndNil(t *testing.T) {
	if cmdutil.WrapUsage(nil) != nil {
		t.Error("WrapUsage(nil) should be nil")
	}
	inner := errors.New("bad flag")
	wrapped := cmdutil.WrapUsage(inner)
	if wrapped.Error() != "bad flag" || !errors.Is(wrapped, inner) {
		t.Errorf("wrapped = %v", wrapped)
	}
	if again := cmdutil.WrapUsage(wrapped); again != wrapped {
		t.Error("WrapUsage should not double-wrap")
	}
}

func TestRequireFlagIsUsageError(t *testing.T) {
	err := cmdutil.RequireFlag("name", "")
	if cmdutil.ExitCode(err) != cmdutil.ExitUsage || err.Error() != "--name is required" {
		t.Errorf("got %v (code %d)", err, cmdutil.ExitCode(err))
	}
	if err := cmdutil.RequireFlag("name", "x"); err != nil {
		t.Errorf("got %v", err)
	}
}
