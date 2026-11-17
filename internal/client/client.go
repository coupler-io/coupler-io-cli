// Package client is a thin HTTP client for the Coupler.io public API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production API host.
const DefaultBaseURL = "https://api.coupler.io"

// UserAgent is sent with every request. main sets it to include the version.
var UserAgent = "coupler-io-cli"

type Client struct {
	httpClient *http.Client
	apiToken   string
	baseURL    string
}

type Option func(*Client)

// WithBaseURL overrides the API host, e.g. for staging or tests.
func WithBaseURL(rawurl string) Option {
	return func(c *Client) {
		if rawurl != "" {
			c.baseURL = strings.TrimRight(rawurl, "/")
		}
	}
}

// WithHTTPClient replaces the underlying http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

func New(apiToken string, opts ...Option) *Client {
	c := &Client{
		// Dataset queries may take up to 60s server side.
		httpClient: &http.Client{Timeout: 90 * time.Second},
		apiToken:   apiToken,
		baseURL:    DefaultBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// BaseURL returns the API host the client talks to.
func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, result interface{}) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return newAPIError(resp, respBody)
	}

	if result != nil && len(bytes.TrimSpace(respBody)) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// Get performs a GET request and decodes the JSON response into result.
// result is usually a *json.RawMessage so commands can print the body as is.
func (c *Client) Get(ctx context.Context, path string, query url.Values, result interface{}) error {
	return c.do(ctx, http.MethodGet, path, query, nil, result)
}

func (c *Client) Post(ctx context.Context, path string, body, result interface{}) error {
	return c.do(ctx, http.MethodPost, path, nil, body, result)
}

func (c *Client) Patch(ctx context.Context, path string, body, result interface{}) error {
	return c.do(ctx, http.MethodPatch, path, nil, body, result)
}

func (c *Client) Put(ctx context.Context, path string, body, result interface{}) error {
	return c.do(ctx, http.MethodPut, path, nil, body, result)
}

func (c *Client) Delete(ctx context.Context, path string, result interface{}) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil, result)
}
