package cmd

import (
	"bytes"
	"errors"
	"syscall"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func TestReport(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		sig        syscall.Signal
		wantCode   int
		wantStderr string
	}{
		{"success", nil, 0, 0, ""},
		{"runtime error", errors.New("boom"), 0, 1, "Error: boom\n"},
		{"usage error", cmdutil.UsageErrorf("--x is required"), 0, 2, "Error: --x is required\n"},
		{"sigint", errors.New("context canceled"), syscall.SIGINT, 130, "Interrupted\n"},
		{"sigterm", nil, syscall.SIGTERM, 143, "Interrupted\n"},
	}
	for _, tc := range cases {
		var stderr bytes.Buffer
		if got := report(&stderr, tc.err, tc.sig); got != tc.wantCode {
			t.Errorf("%s: code = %d, want %d", tc.name, got, tc.wantCode)
		}
		if stderr.String() != tc.wantStderr {
			t.Errorf("%s: stderr = %q, want %q", tc.name, stderr.String(), tc.wantStderr)
		}
	}
}

func TestNotifyContextWithoutSignal(t *testing.T) {
	ctx, received, stop := notifyContext()
	if received() != 0 {
		t.Error("no signal expected yet")
	}
	stop()
	if ctx.Err() == nil {
		t.Error("stop should cancel the context")
	}
	if received() != 0 {
		t.Errorf("stop must not record a signal, got %v", received())
	}
}
