package inworld

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeWireEvent_merged_layers(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"result": map[string]any{
			"transcription": map[string]any{"transcript": "from result", "isFinal": false},
			"voiceProfile": map[string]any{
				"emotion": []any{map[string]any{"label": "neutral", "confidence": 0.95}},
			},
			"voice_profile": map[string]any{
				"emotion": []any{map[string]any{"label": "happy", "confidence": 0.6}},
			},
		},
		"transcription": map[string]any{"transcript": "from root transcription", "isFinal": true},
		"transcript":    map[string]any{"text": "final text", "isFinal": false},
	}
	ev, err := decodeWireEvent(raw)
	require.NoError(t, err)
	got := ev.merged()
	require.Equal(t, "final text", got.Text)
	require.False(t, got.IsFinal)
	require.Len(t, got.VoiceProfile.Emotion, 1)
	require.Equal(t, "happy", got.VoiceProfile.Emotion[0].Label)
}

func TestDecodeWireEvent_resultTranscriptionOnly(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"result": map[string]any{
			"transcription": map[string]any{"transcript": "hello", "isFinal": true},
		},
	}
	ev, err := decodeWireEvent(raw)
	require.NoError(t, err)
	got := ev.merged()
	require.Equal(t, "hello", got.Text)
	require.True(t, got.IsFinal)
}
