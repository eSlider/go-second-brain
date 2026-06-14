// Package neo4j provides Neo4j driver wiring for Edelweiss services.
package neo4j

import "context"

// Config configures a Neo4j driver (bolt URL and credentials).
type Config struct {
	URI      string
	User     string `default:"neo4j"`
	Password string
}

// GetClient returns a reusable Neo4j client.
func (c *Config) GetClient(ctx context.Context) (client *Client, err error) {
	return New(ctx, c)
}
