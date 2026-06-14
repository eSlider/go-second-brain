package neo4j

import (
	"context"
	"fmt"
	"time"

	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client holds a neo4j-go-driver DriverWithContext.
type Client struct {
	Driver driver.DriverWithContext
}

// New connects to Neo4j and returns a reusable client (close with [*Client.Close]).
func New(_ context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("neo4j: Config is nil")
	}
	d, err := driver.NewDriverWithContext(cfg.URI, driver.BasicAuth(cfg.User, cfg.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j: driver: %w", err)
	}
	return &Client{Driver: d}, nil
}

// Close closes the Neo4j driver using a bounded background context.
func (c *Client) Close() error {
	if c == nil || c.Driver == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.Driver.Close(ctx)
}
