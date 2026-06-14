package httpclient

import (
	"context"
	"time"

	"github.com/mcuadros/go-defaults"
)

// Client is a trivial holder exposing Timeout for [*ollama.New] / [*qdrant.New] composition.
type Client struct {
	Config Config
}

// New allocates cfg defaults when missing.
func New(_ context.Context, cfg *Config) (*Client, error) {
	var c Config
	if cfg != nil {
		c = *cfg
	}
	defaults.SetDefaults(&c)
	if c.Timeout <= 0 {
		c.Timeout = 120 * time.Second
	}
	return &Client{Config: c}, nil
}

// Close is a noop.
func (*Client) Close() error { return nil }

// Timeout returns the configured outbound HTTP dial+read deadline budget.
func (c *Client) Timeout() time.Duration { return c.Config.Timeout }
