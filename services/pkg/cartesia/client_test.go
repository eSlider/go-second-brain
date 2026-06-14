package cartesia

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
