package assistant

import "testing"

func TestExtractAudioChunk(t *testing.T) {
	t.Parallel()

	raw := map[string]any{"chunk": map[string]any{"data": "Zm9v"}}
	got, ok := extractAudioChunk(raw)
	if !ok {
		t.Fatalf("expected audio chunk")
	}
	if got != "Zm9v" {
		t.Fatalf("unexpected data: %s", got)
	}
}

func TestParseVoiceProfileBuckets(t *testing.T) {
	t.Parallel()

	raw := []any{
		map[string]any{"label": "adult", "confidence": 0.9},
		map[string]any{"label": "young", "confidence": 0.1},
	}
	got := parseVoiceProfileBuckets(raw)
	if len(got) != 2 {
		t.Fatalf("unexpected buckets count: %d", len(got))
	}
	if got[0].Label != "adult" {
		t.Fatalf("unexpected first label: %s", got[0].Label)
	}
}
