package llm

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError is a typed error returned by a provider when the HTTP call fails
// with a non-200 status. It carries the status code and, when the server
// provides one, the Retry-After hint so the retry layer can wait exactly as
// long as the provider asks instead of guessing.
type APIError struct {
	Provider   string        // "openai" / "anthropic"
	Status     int           // HTTP status code
	Body       string        // raw response body (for logs / detail)
	RetryAfter time.Duration // parsed Retry-After header (0 if absent)
}

func (e *APIError) Error() string {
	// Friendly, human-readable message — no raw JSON dumped at the user.
	switch e.Status {
	case http.StatusTooManyRequests:
		if e.RetryAfter > 0 {
			return fmt.Sprintf("rate limited by %s (429) — retrying in %s", e.Provider, e.RetryAfter.Round(time.Second))
		}
		return fmt.Sprintf("rate limited by %s (429) — backing off and retrying", e.Provider)
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Sprintf("%s auth failed (%d) — check your API key", e.Provider, e.Status)
	case http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return fmt.Sprintf("%s server error (%d) — retrying", e.Provider, e.Status)
	default:
		msg := strings.TrimSpace(e.Body)
		if len(msg) > 200 {
			msg = msg[:200] + "…"
		}
		return fmt.Sprintf("%s %d: %s", e.Provider, e.Status, msg)
	}
}

// RetryAfterHint exposes the server-provided Retry-After delay so the retry
// layer can wait exactly as long as the provider asked. Satisfies the
// (unexported) serverHinted interface in the retry package.
func (e *APIError) RetryAfterHint() time.Duration { return e.RetryAfter }

// Retryable reports whether this API error is safe to retry.
func (e *APIError) Retryable() bool {
	switch e.Status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// newAPIError builds an APIError from an HTTP response, parsing Retry-After.
func newAPIError(provider string, resp *http.Response, body string) *APIError {
	return &APIError{
		Provider:   provider,
		Status:     resp.StatusCode,
		Body:       body,
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}
}

// parseRetryAfter interprets a Retry-After header, which may be either an
// integer number of seconds ("30") or an HTTP date. Returns 0 if unparseable.
func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	// Numeric seconds form.
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	// HTTP-date form.
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}
