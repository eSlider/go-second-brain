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
	System  string `json:"system,omitempty"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Options any    `json:"options,omitempty"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Generate returns the full model output (non-streaming). For models that support it, use GenerateWithSystem.
func (c *Client) Generate(ctx context.Context, model, prompt string) (string, error) {
	return c.GenerateWithSystem(ctx, model, "", prompt)
}

// GenerateWithSystem passes a system prompt to Ollama (behavior rules); empty system is omitted from the request.
func (c *Client) GenerateWithSystem(ctx context.Context, model, system, prompt string) (string, error) {
	req := generateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}
	if strings.TrimSpace(system) != "" {
		req.System = system
	}
	raw, err := c.HTTP.PostRaw(ctx, "/api/generate", req)
	if err != nil {
		return "", err
	}
	var resp generateResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("llm: decode: %w", err)
	}
	return strings.TrimSpace(resp.Response), nil
}
