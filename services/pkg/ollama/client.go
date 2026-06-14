package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"git.produktor.io/edelweiss/docs/services/pkg/httpjson"
)

// Client wraps Ollama /api/embeddings and /api/generate HTTP calls.
type Client struct {
	HTTP *httpjson.Client
}

// New constructs an Ollama client using cfg.URL (trimmed).
func New(_ context.Context, cfg *Config) (*Client, error) {
	if cfg == nil || strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("ollama: URL is required")
	}
	t := cfg.Timeout
	if t <= 0 {
		t = 120 * time.Second
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	return &Client{HTTP: httpjson.New(base, t)}, nil
}

// Close releases resources (currently no pooled resources; satisfies lifecycle pattern).
func (c *Client) Close() error { return nil }

type embeddingsRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// Embed returns a vector from POST /api/embeddings.
func (c *Client) Embed(ctx context.Context, model, text string) ([]float32, error) {
	raw, err := c.HTTP.PostRaw(ctx, "/api/embeddings", embeddingsRequest{Model: model, Prompt: text})
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("ollama: decode envelope: %w", err)
	}
	embRaw, ok := m["embedding"]
	if !ok {
		return nil, fmt.Errorf("ollama: missing embedding field")
	}
	var flat []float64
	if err := json.Unmarshal(embRaw, &flat); err == nil && len(flat) > 0 {
		return toFloat32(flat), nil
	}
	var nested [][]float64
	if err := json.Unmarshal(embRaw, &nested); err == nil && len(nested) > 0 && len(nested[0]) > 0 {
		return toFloat32(nested[0]), nil
	}
	return nil, fmt.Errorf("ollama: unsupported embedding shape")
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

// Generate returns model output via POST /api/generate (non-streaming).
func (c *Client) Generate(ctx context.Context, model, prompt string) (string, error) {
	return c.GenerateWithSystem(ctx, model, "", prompt)
}

// GenerateWithSystem passes an optional system prompt.
func (c *Client) GenerateWithSystem(ctx context.Context, model, system, prompt string) (string, error) {
	req := generateRequest{Model: model, Prompt: prompt, Stream: false}
	if strings.TrimSpace(system) != "" {
		req.System = system
	}
	raw, err := c.HTTP.PostRaw(ctx, "/api/generate", req)
	if err != nil {
		return "", err
	}
	var resp generateResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("ollama: decode generate: %w", err)
	}
	return strings.TrimSpace(resp.Response), nil
}

func toFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}
