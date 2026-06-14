package botcmd

import "context"

// Client resolves how users invoke `!`-style prefixes in Matrix rooms.
type Client struct {
	cfg Config
}

// New binds cfg onto a client accessor.
func New(_ context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return &Client{}, nil
	}
	c := *cfg
	return &Client{cfg: c}, nil
}

// Close is a noop.
func (*Client) Close() error { return nil }

// CommandPrefix returns the configured textual prefix prior to trimming.
func (c *Client) CommandPrefix() string { return c.cfg.Command.Prefix }
