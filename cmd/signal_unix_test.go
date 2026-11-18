//go:build unix

package cmd

import (
	"context"
	"syscall"
	"testing"
	"time"
)

func TestNotifyContextRecordsSignal(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		ctx, received, stop := notifyContext()
		if err := syscall.Kill(syscall.Getpid(), sig); err != nil {
			stop()
			t.Fatal(err)
		}
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
			stop()
			t.Fatalf("%v: context was not cancelled", sig)
		}
		if got := received(); got != sig {
			t.Errorf("received = %v, want %v", got, sig)
		}
		if ctx.Err() != context.Canceled {
			t.Errorf("ctx.Err() = %v", ctx.Err())
		}
		stop()
	}
}
