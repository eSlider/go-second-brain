package embedding

import (
	"context"
	"fmt"

	"github.com/eSlider/go-second-brain/services/pkg/ollama"
)

// Client projects text into embeddings using a pinned model on a shared [*ollama.Client].
type Client struct {
	llm    *ollama.Client
	Config Config
}

// New attaches an Ollama HTTP client plus embedding.Config.
func New(_ context.Context, llm *ollama.Client, cfg *Config) (*Client, error) {
	if llm == nil || cfg == nil {
		return nil, fmt.Errorf("embedding: ollama.Client and embedding.Config are required")
	}
	cc := *cfg
	return &Client{llm: llm, Config: cc}, nil
}

// Close is a noop; closing the [*ollama.Client] lifecycle is callers' duty.
func (*Client) Close() error { return nil }

// Embed tokenizes+pools via Ollama /api/embeddings for this client's model string.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	return c.llm.Embed(ctx, c.Config.Model, text)
}
