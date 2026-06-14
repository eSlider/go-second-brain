package assist

import "context"

// Client exposes assistant runtime knobs as methods (paths, pacing, PCM rate).
type Client struct {
	Config Config
}

// New binds cfg snapshot.
func New(_ context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return &Client{}, nil
	}
	c := *cfg
	return &Client{Config: c}, nil
}

// Close is a noop.
func (*Client) Close() error { return nil }
