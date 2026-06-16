package cartesia

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	wsURL   = "wss://api.cartesia.ai/tts/websocket"
	version = "2026-03-01"
)

// Client streams TTS audio over Cartesia WebSockets (opens a socket per [*Client.Stream] call).
type Client struct {
	cfg Config
}

// New validates cfg and returns a client bound to Cartesia wsURL.
func New(_ context.Context, cfg *Config) (*Client, error) {
	if cfg == nil || cfg.API.Key == "" {
		return nil, fmt.Errorf("cartesia: API key is required")
	}
	if cfg.Voice.ID == "" {
		return nil, fmt.Errorf("cartesia: voice id is required")
	}
	if cfg.Model.ID == "" {
		cfg.Model.ID = "sonic-3.5"
	}
	if cfg.Language == "" {
		cfg.Language = "ru"
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 16000
	}
	cfgCopy := *cfg
	return &Client{cfg: cfgCopy}, nil
}

// Close is a noop for symmetry (each Stream owns its websocket lifecycle).
func (c *Client) Close() error { return nil }

// Stream sends transcript text for synthesis and writes decoded PCM chunks to out.
func (c *Client) Stream(ctx context.Context, text string, out chan<- []byte) error {
	if text == "" {
		return nil
	}
	headers := http.Header{}
	headers.Set("Cartesia-Version", version)
	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("cartesia: parse url: %w", err)
	}
	q := u.Query()
	q.Set("api_key", c.cfg.API.Key)
	q.Set("cartesia_version", version)
	u.RawQuery = q.Encode()
	conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return fmt.Errorf("cartesia: dial: %w", err)
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "done")
	}()
	req := map[string]any{
		"model_id": c.cfg.Model.ID,
		"voice": map[string]any{
			"mode": "id",
			"id":   c.cfg.Voice.ID,
		},
		"transcript": text,
		"language":   c.cfg.Language,
		"output_format": map[string]any{
			"container":   "raw",
			"encoding":    "pcm_s16le",
			"sample_rate": c.cfg.SampleRate,
		},
		"stream":     true,
		"context_id": randomContextID(),
		"continue":   false,
	}
	if err := wsjson.Write(ctx, conn, req); err != nil {
		return fmt.Errorf("cartesia: send request: %w", err)
	}
	for {
		var raw map[string]any
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			return fmt.Errorf("cartesia: read: %w", err)
		}
		if done, ok := raw["done"].(bool); ok && done {
			return nil
		}
		chunkB64, ok := extractAudioChunk(raw)
		if !ok || chunkB64 == "" {
			continue
		}
		chunk, err := base64.StdEncoding.DecodeString(chunkB64)
		if err != nil {
			return fmt.Errorf("cartesia: decode audio chunk: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- chunk:
		}
	}
}

func extractAudioChunk(raw map[string]any) (string, bool) {
	if v, ok := raw["data"].(string); ok {
		return v, true
	}
	if v, ok := raw["audio"].(string); ok {
		return v, true
	}
	if obj, ok := raw["chunk"].(map[string]any); ok {
		if v, ok := obj["data"].(string); ok {
			return v, true
		}
	}
	return "", false
}

func randomContextID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
