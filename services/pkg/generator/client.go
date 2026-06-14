package generator

import (
	"context"
	"fmt"

	"git.produktor.io/edelweiss/docs/services/pkg/ollama"
)

// Client performs non-stream completion using a pinned model on [*ollama.Client].
type Client struct {
	llm    *ollama.Client
	Config Config
}

// New attaches an existing Ollama client.
func New(_ context.Context, llm *ollama.Client, cfg *Config) (*Client, error) {
	if llm == nil || cfg == nil {
		return nil, fmt.Errorf("generator: ollama.Client and generator.Config are required")
	}
	cc := *cfg
	return &Client{llm: llm, Config: cc}, nil
}

// Close is a noop; close the [*ollama.Client] from your composition root instead.
func (*Client) Close() error { return nil }

// GenerateWithSystem proxies to Ollama /api/generate with optional system preamble.
func (c *Client) GenerateWithSystem(ctx context.Context, system, prompt string) (string, error) {
	return c.llm.GenerateWithSystem(ctx, c.Config.Model, system, prompt)
}
