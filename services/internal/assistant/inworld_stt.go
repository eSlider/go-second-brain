package assistant

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const inworldSTTURL = "wss://api.inworld.ai/stt/v1/transcribe:streamBidirectional"

type VoiceProfileBucket struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

type VoiceProfile struct {
	Age        []VoiceProfileBucket `json:"age"`
	Accent     []VoiceProfileBucket `json:"accent"`
	Emotion    []VoiceProfileBucket `json:"emotion"`
	VocalStyle []VoiceProfileBucket `json:"vocal_style"`
	Pitch      []VoiceProfileBucket `json:"pitch"`
}

type STTEvent struct {
	Text         string       `json:"text"`
	IsFinal      bool         `json:"isFinal"`
	VoiceProfile VoiceProfile `json:"voice_profile"`
}

type InworldSTT struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func DialInworldSTT(ctx context.Context, apiKey string, modelID string) (*InworldSTT, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("assistant stt: api key is required")
	}
	if modelID == "" || modelID == "base" {
		modelID = "inworld/inworld-stt-1"
	}
	headers := http.Header{}
	headers.Set("Authorization", "Basic "+apiKey)
	conn, _, err := websocket.Dial(ctx, inworldSTTURL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("assistant stt: dial: %w", err)
	}
	client := &InworldSTT{conn: conn}
	if err := client.sendConfig(ctx, modelID); err != nil {
		_ = conn.Close(websocket.StatusInternalError, "config failed")
		return nil, err
	}
	return client, nil
}

func (c *InworldSTT) sendConfig(ctx context.Context, modelID string) error {
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
		return fmt.Errorf("assistant stt: send config: %w", err)
	}
	return nil
}

func (c *InworldSTT) SendPCMChunk(ctx context.Context, pcm []byte) error {
	if len(pcm) == 0 {
		return nil
	}
	msg := map[string]any{
		"audio_chunk": map[string]any{
			"content": base64.StdEncoding.EncodeToString(pcm),
		},
	}
	if err := c.writeJSON(ctx, msg); err != nil {
		return fmt.Errorf("assistant stt: send chunk: %w", err)
	}
	return nil
}

func (c *InworldSTT) SendEndTurn(ctx context.Context) error {
	msg := map[string]any{"end_turn": map[string]any{}}
	if err := c.writeJSON(ctx, msg); err != nil {
		return fmt.Errorf("assistant stt: send end turn: %w", err)
	}
	return nil
}

func (c *InworldSTT) writeJSON(ctx context.Context, msg map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return wsjson.Write(ctx, c.conn, msg)
}

func (c *InworldSTT) ReadEvent(ctx context.Context) (STTEvent, error) {
	var raw map[string]any
	if err := wsjson.Read(ctx, c.conn, &raw); err != nil {
		return STTEvent{}, fmt.Errorf("assistant stt: read: %w", err)
	}
	if msg, ok := raw["message"].(string); ok {
		if code, ok := raw["code"].(float64); ok && code > 0 {
			return STTEvent{}, fmt.Errorf("assistant stt: api error code=%.0f message=%s", code, msg)
		}
	}
	var out STTEvent
	if result, ok := raw["result"].(map[string]any); ok {
		if tr, ok := result["transcription"].(map[string]any); ok {
			if v, ok := tr["transcript"].(string); ok {
				out.Text = v
			}
			if v, ok := tr["isFinal"].(bool); ok {
				out.IsFinal = v
			}
		}
		if vp, ok := result["voiceProfile"].(map[string]any); ok {
			out.VoiceProfile = parseVoiceProfile(vp)
		}
		if vp, ok := result["voice_profile"].(map[string]any); ok {
			out.VoiceProfile = parseVoiceProfile(vp)
		}
	}
	if tr, ok := raw["transcription"].(map[string]any); ok {
		if v, ok := tr["transcript"].(string); ok {
			out.Text = v
		}
		if v, ok := tr["isFinal"].(bool); ok {
			out.IsFinal = v
		}
	}
	if tr, ok := raw["transcript"].(map[string]any); ok {
		if v, ok := tr["text"].(string); ok {
			out.Text = v
		}
		if v, ok := tr["isFinal"].(bool); ok {
			out.IsFinal = v
		}
	}
	return out, nil
}

func parseVoiceProfile(raw map[string]any) VoiceProfile {
	return VoiceProfile{
		Age:        parseVoiceProfileBuckets(raw["age"]),
		Accent:     parseVoiceProfileBuckets(raw["accent"]),
		Emotion:    parseVoiceProfileBuckets(raw["emotion"]),
		VocalStyle: parseVoiceProfileBuckets(raw["vocal_style"]),
		Pitch:      parseVoiceProfileBuckets(raw["pitch"]),
	}
}

func parseVoiceProfileBuckets(raw any) []VoiceProfileBucket {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]VoiceProfileBucket, 0, len(items))
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		var bucket VoiceProfileBucket
		if label, ok := m["label"].(string); ok {
			bucket.Label = label
		}
		if conf, ok := m["confidence"].(float64); ok {
			bucket.Confidence = conf
		}
		out = append(out, bucket)
	}
	return out
}

func (c *InworldSTT) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close(websocket.StatusNormalClosure, "bye")
}
