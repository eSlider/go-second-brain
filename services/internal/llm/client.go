// Package llm calls Ollama /api/generate.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.produktor.io/edelweiss/docs/services/internal/httpjson"
)

// Client generates text from a prompt.
type Client struct {
	HTTP *httpjson.Client
}

// New creates an LLM client for a base Ollama URL.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{HTTP: httpjson.New(baseURL, timeout)}
}

type generateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Options any    `json:"options,omitempty"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Generate returns the full model output (non-streaming).
func (c *Client) Generate(ctx context.Context, model, prompt string) (string, error) {
	raw, err := c.HTTP.PostRaw(ctx, "/api/generate", generateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", err
	}
	var resp generateResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("llm: decode: %w", err)
	}
	return strings.TrimSpace(resp.Response), nil
}
