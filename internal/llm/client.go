package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gjkjk/zed/internal/cache"
	"github.com/gjkjk/zed/internal/config"
)

// Client is a provider-agnostic LLM interface. Implementations stream events
// over the returned channel and close it when finished.
type Client interface {
	// Name returns the provider identifier.
	Name() string
	// Stream sends a request and returns a channel of streaming events.
	Stream(ctx context.Context, req Request) (<-chan StreamEvent, error)
	// Complete sends a request and returns the full response (non-streaming).
	Complete(ctx context.Context, req Request) (Response, error)
}

// New builds a Client from config, selecting the correct provider.
func New(cfg *config.Config) (Client, error) {
	var inner Client
	switch cfg.Provider {
	case "anthropic", "":
		inner = newAnthropic(cfg)
	case "openai":
		inner = newOpenAI(cfg)
	default:
		return nil, fmt.Errorf("unknown provider %q", cfg.Provider)
	}
	return inner, nil
}

// WithCache wraps a Client so that non-streaming (Complete) calls are cached
// with a TTL. Streaming calls pass through uncached.
func WithCache(inner Client, c *cache.Cache) Client {
	return &cachedClient{inner: inner, cache: c}
}

// cachedClient wraps a Client with a response cache for Complete calls.
type cachedClient struct {
	inner Client
	cache *cache.Cache
}

func (c *cachedClient) Name() string { return c.inner.Name() }

func (c *cachedClient) Stream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	return c.inner.Stream(ctx, req)
}

func (c *cachedClient) Complete(ctx context.Context, req Request) (Response, error) {
	msgBytes, _ := json.Marshal(req.Messages)
	key := fmt.Sprintf("%s|%s|%s|%s", c.Name(), req.Model, req.System, string(msgBytes))
	val, err := c.cache.GetOrCompute(key, 5*time.Minute, func() (string, error) {
		resp, err := c.inner.Complete(ctx, req)
		if err != nil {
			return "", err
		}
		encoded, err := json.Marshal(resp)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	})
	if err != nil {
		return Response{}, err
	}
	var resp Response
	if err := json.Unmarshal([]byte(val), &resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

// NewCached creates a Client from config with an optional cache layer.
func NewCached(cfg *config.Config, c *cache.Cache) (Client, error) {
	inner, err := New(cfg)
	if err != nil {
		return nil, err
	}
	if c != nil {
		return WithCache(inner, c), nil
	}
	return inner, nil
}
