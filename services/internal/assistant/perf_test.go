package assistant

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPerfLogger_Event(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "logs", "performance.jsonl")
	p, err := NewPerfLogger(path)
	if err != nil {
		t.Fatalf("new perf logger: %v", err)
	}
	defer func() {
		_ = p.Close()
	}()

	if err := p.Event("tts_first_byte", map[string]any{"latency_ms": 12}); err != nil {
		t.Fatalf("event: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "\"event\":\"tts_first_byte\"") {
		t.Fatalf("missing event in log: %s", s)
	}
	if !strings.Contains(s, "\"session_id\"") {
		t.Fatalf("missing session_id in log: %s", s)
	}
}
