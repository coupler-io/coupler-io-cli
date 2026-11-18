// Package cmdtest provides helpers for testing commands against a fake API.
package cmdtest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/coupler-io/coupler-io-cli/internal/client"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

// Env is a Factory wired to a fake API server plus captured output.
type Env struct {
	Factory *cmdutil.Factory
	URL     string // base URL of the fake API
	Out     *bytes.Buffer
	ErrOut  *bytes.Buffer
}

// Setup starts a fake API running handler and returns a Factory pointing at
// it. The Factory gets its own viper instance, preloaded with workspace "42"
// and table output.
func Setup(t *testing.T, handler http.HandlerFunc) *Env {
	t.Helper()

	v := viper.New()
	v.Set("workspace-id", "42")
	v.Set("output", "table")

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	env := &Env{URL: srv.URL, Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}}
	env.Factory = &cmdutil.Factory{
		Viper: v,
		IOStreams: &cmdutil.IOStreams{
			In:     &bytes.Buffer{},
			Out:    env.Out,
			ErrOut: env.ErrOut,
		},
		ClientOverride: client.New("test-token", client.WithBaseURL(srv.URL)),
	}
	return env
}

// Run executes cmd with args and returns its error.
func Run(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	return cmd.Execute()
}

// JSON writes v as a JSON response.
func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// DecodeBody decodes the request body into a generic map.
func DecodeBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal body %q: %v", data, err)
	}
	return payload
}
