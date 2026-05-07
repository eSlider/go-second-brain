package assistant

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const cartesiaTTSURL = "wss://api.cartesia.ai/tts/websocket"
const cartesiaVersion = "2026-03-01"

type CartesiaTTS struct {
	apiKey   string
	modelID  string
	voiceID  string
	language string
}

func NewCartesiaTTS(apiKey string, modelID string, voiceID string, language string) (*CartesiaTTS, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("assistant tts: api key is required")
	}
	if modelID == "" {
		modelID = "sonic-3.5"
	}
	if voiceID == "" {
		return nil, fmt.Errorf("assistant tts: voice id is required")
	}
	if language == "" {
		language = "ru"
	}
	return &CartesiaTTS{
		apiKey:   apiKey,
		modelID:  modelID,
		voiceID:  voiceID,
		language: language,
	}, nil
}

// Stream sends text for synthesis and emits decoded PCM chunks.
func (c *CartesiaTTS) Stream(ctx context.Context, text string, out chan<- []byte) error {
	if text == "" {
		return nil
	}
	headers := http.Header{}
	headers.Set("Cartesia-Version", cartesiaVersion)
	u, err := url.Parse(cartesiaTTSURL)
	if err != nil {
		return fmt.Errorf("assistant tts: parse url: %w", err)
	}
	q := u.Query()
	q.Set("api_key", c.apiKey)
	q.Set("cartesia_version", cartesiaVersion)
	u.RawQuery = q.Encode()
	conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return fmt.Errorf("assistant tts: dial: %w", err)
	}
	defer func() {
		_ = conn.Close(websocket.StatusNormalClosure, "done")
	}()
	req := map[string]any{
		"model_id": c.modelID,
		"voice": map[string]any{
			"mode": "id",
			"id":   c.voiceID,
		},
		"transcript": text,
		"language":   c.language,
		"output_format": map[string]any{
			"container":   "raw",
			"encoding":    "pcm_s16le",
			"sample_rate": 16000,
		},
		"stream":     true,
		"context_id": randomID(),
		"continue":   false,
	}
	if err := wsjson.Write(ctx, conn, req); err != nil {
		return fmt.Errorf("assistant tts: send request: %w", err)
	}
	for {
		var raw map[string]any
		if err := wsjson.Read(ctx, conn, &raw); err != nil {
			return fmt.Errorf("assistant tts: read: %w", err)
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
			return fmt.Errorf("assistant tts: decode audio chunk: %w", err)
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
