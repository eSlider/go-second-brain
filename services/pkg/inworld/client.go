package inworld

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const transcribeWS = "wss://api.inworld.ai/stt/v1/transcribe:streamBidirectional"

// Client is a bidirectional streaming STT session (one websocket for the lifetime of the client).
type Client struct {
	conn *websocket.Conn
	mu   sync.Mutex
	cfg  Config
}

// New dials Inworld STT with cfg.API.Key as Basic auth credential and configures the session.
func New(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil || cfg.API.Key == "" {
		return nil, fmt.Errorf("inworld: API key is required")
	}
	modelID := cfg.STT.Model
	if modelID == "" || modelID == "base" {
		modelID = "inworld/inworld-stt-1"
	}
	headers := http.Header{}
	headers.Set("Authorization", "Basic "+cfg.API.Key)
	conn, _, err := websocket.Dial(ctx, transcribeWS, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("inworld: dial: %w", err)
	}
	c := &Client{conn: conn, cfg: *cfg}
	if err := c.sendConfig(ctx, modelID); err != nil {
		_ = conn.Close(websocket.StatusInternalError, "config failed")
		return nil, err
	}
	return c, nil
}

func (c *Client) sendConfig(ctx context.Context, modelID string) error {
	msg := map[string]any{
		"transcribe_config": map[string]any{
			"modelId":          modelID,
			"audioEncoding":    "LINEAR16",
			"sampleRateHertz":  16000,
			"numberOfChannels": 1,
			"voiceProfileConfig": map[string]any{
				"enableVoiceProfile": true,
				"topN":               3,
			},
			"inworldSttV1Config": map[string]any{
				"vadThreshold": 0.5,
			},
		},
	}
	if err := c.writeJSON(ctx, msg); err != nil {
		return fmt.Errorf("inworld: send config: %w", err)
	}
	return nil
}

// SendPCMChunk sends one LINEAR16 mono PCM window.
func (c *Client) SendPCMChunk(ctx context.Context, pcm []byte) error {
	if len(pcm) == 0 {
		return nil
	}
	msg := map[string]any{
		"audio_chunk": map[string]any{
			"content": base64.StdEncoding.EncodeToString(pcm),
		},
	}
	if err := c.writeJSON(ctx, msg); err != nil {
		return fmt.Errorf("inworld: send chunk: %w", err)
	}
	return nil
}

// SendEndTurn notifies the recognizer that the user finished an utterance.
func (c *Client) SendEndTurn(ctx context.Context) error {
	msg := map[string]any{"end_turn": map[string]any{}}
	if err := c.writeJSON(ctx, msg); err != nil {
		return fmt.Errorf("inworld: send end turn: %w", err)
	}
	return nil
}

func (c *Client) writeJSON(ctx context.Context, msg map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return wsjson.Write(ctx, c.conn, msg)
}

// ReadEvent blocks for the next STT message.
func (c *Client) ReadEvent(ctx context.Context) (Event, error) {
	var raw map[string]any
	if err := wsjson.Read(ctx, c.conn, &raw); err != nil {
		return Event{}, fmt.Errorf("inworld: read: %w", err)
	}
	ev, err := decodeWireEvent(raw)
	if err != nil {
		return Event{}, err
	}
	if msg := ev.Message; msg != "" {
		if code := ev.Code; code > 0 {
			return Event{}, fmt.Errorf("inworld: api error code=%.0f message=%s", code, msg)
		}
	}
	return ev.merged(), nil
}

// Close closes the websocket.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close(websocket.StatusNormalClosure, "bye")
}
