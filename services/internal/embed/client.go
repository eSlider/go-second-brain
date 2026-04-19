// Package embed calls Ollama /api/embeddings.
package embed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"git.produktor.io/edelweiss/docs/services/internal/httpjson"
)

// Client embeds text into a float32 vector.
type Client struct {
	HTTP *httpjson.Client
}

// New creates an embed client for a base Ollama URL (e.g. http://127.0.0.1:11434).
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{HTTP: httpjson.New(baseURL, timeout)}
}

type embeddingsRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// Embed returns a float32 slice from Ollama.
func (c *Client) Embed(ctx context.Context, model, text string) ([]float32, error) {
	raw, err := c.HTTP.PostRaw(ctx, "/api/embeddings", embeddingsRequest{Model: model, Prompt: text})
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("embed: decode envelope: %w", err)
	}
	embRaw, ok := m["embedding"]
	if !ok {
		return nil, fmt.Errorf("embed: missing embedding field")
	}
	var flat []float64
	if err := json.Unmarshal(embRaw, &flat); err == nil && len(flat) > 0 {
		return toFloat32(flat), nil
	}
	var nested [][]float64
	if err := json.Unmarshal(embRaw, &nested); err == nil && len(nested) > 0 && len(nested[0]) > 0 {
		return toFloat32(nested[0]), nil
	}
	return nil, fmt.Errorf("embed: unsupported embedding shape")
}

func toFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}
