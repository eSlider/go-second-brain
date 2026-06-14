package documents

import "context"

// Client exposes loaded document path settings for helpers and tests.
type Client struct {
	cfg Config
}

// New returns a trivial client wrapping cfg (copy).
func New(_ context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		z := Config{}
		return &Client{cfg: z}, nil
	}
	c := *cfg
	return &Client{cfg: c}, nil
}

// Close satisfies the shared lifecycle pattern for config-backed clients.
func (*Client) Close() error { return nil }

// Root returns the corpus root directory.
func (c *Client) Root() string { return c.cfg.Root }
