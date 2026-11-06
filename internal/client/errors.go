package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError is returned for any response with status >= 400.
type APIError struct {
	StatusCode int
	Message    string
	// RetryAfter is the Retry-After header in seconds, set on 429 responses.
	RetryAfter string
	// Body is the raw response body, kept for debugging and --output json.
	Body []byte
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("API error %d", e.StatusCode)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	switch e.StatusCode {
	case http.StatusUnauthorized:
		msg += " (check your API token, or run `coupler configure`)"
	case http.StatusForbidden:
		msg += " (check workspace access and that your plan includes the public API)"
	case http.StatusTooManyRequests:
		if e.RetryAfter != "" {
			msg += fmt.Sprintf(" (retry in %ss)", e.RetryAfter)
		}
	}
	return msg
}

func newAPIError(resp *http.Response, body []byte) *APIError {
	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    extractMessage(body),
		RetryAfter: resp.Header.Get("Retry-After"),
		Body:       body,
	}
}

// extractMessage understands {"error":{"message":"..."}}, {"error":"..."},
// {"message":"..."} and falls back to the (truncated) raw body.
func extractMessage(body []byte) string {
	var payload struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if json.Unmarshal(body, &payload) == nil {
		if len(payload.Error) > 0 {
			var nested struct {
				Message string `json:"message"`
			}
			if json.Unmarshal(payload.Error, &nested) == nil && nested.Message != "" {
				return nested.Message
			}
			var s string
			if json.Unmarshal(payload.Error, &s) == nil && s != "" {
				return s
			}
		}
		if payload.Message != "" {
			return payload.Message
		}
	}

	text := strings.TrimSpace(string(body))
	if len(text) > 500 {
		text = text[:500] + "..."
	}
	return text
}
