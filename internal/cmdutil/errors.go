package cmdutil

import (
	"errors"
	"fmt"
)

// Exit codes follow Unix conventions. Termination by signal N is reported as
// 128+N by the caller (see cmd.Execute).
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
)

// UsageError marks an error caused by invalid flags, arguments or other user
// input, as opposed to a runtime failure. It maps to ExitUsage.
type UsageError struct {
	Err error
}

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

// UsageErrorf builds a UsageError from a format string.
func UsageErrorf(format string, a ...any) error {
	return &UsageError{Err: fmt.Errorf(format, a...)}
}

// WrapUsage marks err as a usage error. It returns nil for a nil error and
// leaves errors that are already usage errors untouched.
func WrapUsage(err error) error {
	if err == nil {
		return nil
	}
	var ue *UsageError
	if errors.As(err, &ue) {
		return err
	}
	return &UsageError{Err: err}
}

// ExitCode maps a command error to a process exit code.
func ExitCode(err error) int {
	var ue *UsageError
	switch {
	case err == nil:
		return ExitOK
	case errors.As(err, &ue):
		return ExitUsage
	default:
		return ExitError
	}
}
