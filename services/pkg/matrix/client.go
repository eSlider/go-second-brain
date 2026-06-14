package matrix

import "context"

// Client holds Matrix bot connectivity settings resolved from Config.
type Client struct {
	Config Config
}

// New validates nothing beyond cfg copy; outbound Matrix sessions use other libraries.
func New(_ context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return &Client{}, nil
	}
	c := *cfg
	return &Client{Config: c}, nil
}

// Close is a noop (no pooled connections at this abstraction layer).
func (*Client) Close() error { return nil }
