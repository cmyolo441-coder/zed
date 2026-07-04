// Package retry provides resilient execution of operations that may transiently
// fail, using exponential backoff with jitter. It is used to wrap LLM API calls
// so temporary rate limits or network blips don't abort an agent run.
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// Policy configures retry behavior.
type Policy struct {
	MaxAttempts int           // total attempts including the first (min 1)
	BaseDelay   time.Duration // initial backoff
	MaxDelay    time.Duration // cap on backoff
	Multiplier  float64       // exponential growth factor
	Jitter      float64       // 0..1 fraction of random jitter
}

// DefaultPolicy is a sensible policy for HTTP API calls. Tuned to ride out
// short rate-limit windows (429): up to 8 attempts with backoff capped at 60s,
// which is also high enough to honor most provider Retry-After hints.
func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts: 8,
		BaseDelay:   1 * time.Second,
		MaxDelay:    60 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.3,
	}
}

// RetryableError marks an error as safe to retry.
type RetryableError struct{ Err error }

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

// Retryable wraps an error to force it to be retried.
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// Permanent wraps an error to force it to NOT be retried.
type Permanent struct{ Err error }

func (e *Permanent) Error() string { return e.Err.Error() }
func (e *Permanent) Unwrap() error { return e.Err }

// PermanentError marks an error as non-retryable.
func PermanentError(err error) error {
	if err == nil {
		return nil
	}
	return &Permanent{Err: err}
}

// OnRetry is an optional callback invoked before each retry sleep.
type OnRetry func(attempt int, delay time.Duration, err error)

// serverHinted is satisfied by errors (e.g. llm.APIError) that carry a
// server-provided Retry-After hint. When present, we wait exactly that long
// instead of using the computed exponential backoff — the server knows best.
type serverHinted interface {
	RetryAfterHint() time.Duration
}

// Do runs fn according to the policy, retrying on transient failures.
func Do(ctx context.Context, p Policy, onRetry OnRetry, fn func() error) error {
	if p.MaxAttempts < 1 {
		p.MaxAttempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= p.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err

		if !shouldRetry(err) || attempt == p.MaxAttempts {
			return unwrapControl(err)
		}

		delay := p.backoff(attempt)
		// If the server told us exactly how long to wait (Retry-After), honor
		// it — but never below our computed backoff, so jitter still spreads
		// concurrent retries.
		var hinted serverHinted
		if errors.As(err, &hinted) {
			if hint := hinted.RetryAfterHint(); hint > delay {
				delay = hint
				if delay > p.MaxDelay {
					delay = p.MaxDelay
				}
			}
		}
		if onRetry != nil {
			onRetry(attempt, delay, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return unwrapControl(lastErr)
}

// DoValue is like Do but returns a value from the operation.
func DoValue[T any](ctx context.Context, p Policy, onRetry OnRetry, fn func() (T, error)) (T, error) {
	var result T
	err := Do(ctx, p, onRetry, func() error {
		v, e := fn()
		if e != nil {
			return e
		}
		result = v
		return nil
	})
	return result, err
}

// backoff computes the delay for a given attempt with jitter.
func (p Policy) backoff(attempt int) time.Duration {
	d := float64(p.BaseDelay) * math.Pow(p.Multiplier, float64(attempt-1))
	if d > float64(p.MaxDelay) {
		d = float64(p.MaxDelay)
	}
	if p.Jitter > 0 {
		j := d * p.Jitter
		d = d - j + rand.Float64()*2*j
	}
	if d < 0 {
		d = 0
	}
	return time.Duration(d)
}

// shouldRetry decides whether an error is transient.
func shouldRetry(err error) bool {
	var perm *Permanent
	if errors.As(err, &perm) {
		return false
	}
	var retry *RetryableError
	if errors.As(err, &retry) {
		return true
	}
	// Heuristics on error text for HTTP status codes and network conditions.
	msg := strings.ToLower(err.Error())
	transient := []string{
		"429", "500", "502", "503", "504",
		"timeout", "timed out", "temporarily",
		"connection reset", "connection refused",
		"connection attempt failed",
		"forcibly closed", "failed to respond",
		"eof", "rate limit", "overloaded",
		"broken pipe", "use of closed",
	}
	for _, t := range transient {
		if strings.Contains(msg, t) {
			return true
		}
	}
	return false
}

// StatusRetryable reports whether an HTTP status code should be retried.
func StatusRetryable(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

func unwrapControl(err error) error {
	var perm *Permanent
	if errors.As(err, &perm) {
		return perm.Err
	}
	var retry *RetryableError
	if errors.As(err, &retry) {
		return retry.Err
	}
	return err
}
