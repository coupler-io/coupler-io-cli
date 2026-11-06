package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/client"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return client.New("secret-token", client.WithBaseURL(srv.URL))
}

func TestGetSendsBearerAuthAndQuery(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		if r.URL.Path != "/v1/workspaces" || r.URL.Query().Get("q") != "a b" {
			t.Errorf("unexpected URL %s", r.URL)
		}
		w.Write([]byte(`{"workspaces":[{"id":1}]}`))
	})

	var body json.RawMessage
	if err := c.Get(context.Background(), "/v1/workspaces", url.Values{"q": {"a b"}}, &body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"workspaces"`) {
		t.Errorf("body = %s", body)
	}
}

func TestPostSendsJSONBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["run_mode"] != "full" {
			t.Errorf("payload = %v", payload)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	var result json.RawMessage
	if err := c.Post(context.Background(), "/x", map[string]string{"run_mode": "full"}, &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for 204, got %s", result)
	}
}

func TestAPIErrorParsing(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		retryAfter string
		wantMsg    string
		wantIn     string
	}{
		{"nested message", 404, `{"error":{"message":"TransformationSet not found"}}`, "", "TransformationSet not found", "API error 404: TransformationSet not found"},
		{"string error", 400, `{"error":"bad input"}`, "", "bad input", "bad input"},
		{"top-level message", 422, `{"message":"nope"}`, "", "nope", "nope"},
		{"plain text", 502, `upstream exploded`, "", "upstream exploded", "upstream exploded"},
		{"unauthorized hint", 401, `{"error":{"message":"invalid token"}}`, "", "invalid token", "coupler configure"},
		{"rate limited", 429, `{"error":{"message":"API rate limit exceeded."}}`, "60", "API rate limit exceeded.", "retry in 60s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if tt.retryAfter != "" {
					w.Header().Set("Retry-After", tt.retryAfter)
				}
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			})

			err := c.Get(context.Background(), "/x", nil, nil)
			var apiErr *client.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *APIError, got %v", err)
			}
			if apiErr.StatusCode != tt.status || apiErr.Message != tt.wantMsg {
				t.Errorf("got status=%d message=%q", apiErr.StatusCode, apiErr.Message)
			}
			if !strings.Contains(apiErr.Error(), tt.wantIn) {
				t.Errorf("Error() = %q, want it to contain %q", apiErr.Error(), tt.wantIn)
			}
		})
	}
}

func TestBaseURLTrailingSlashTrimmed(t *testing.T) {
	c := client.New("t", client.WithBaseURL("https://example.test/"))
	if c.BaseURL() != "https://example.test" {
		t.Errorf("BaseURL = %q", c.BaseURL())
	}
}

func TestEmptyBaseURLKeepsDefault(t *testing.T) {
	c := client.New("t", client.WithBaseURL(""))
	if c.BaseURL() != client.DefaultBaseURL {
		t.Errorf("BaseURL = %q", c.BaseURL())
	}
}
