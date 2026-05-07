package assistant

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMediaStore_WriteAll(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := NewMediaStore(dir)
	if err != nil {
		t.Fatalf("new media store: %v", err)
	}
	path := store.NewFilePath("pcm")
	if !strings.HasSuffix(path, ".pcm") {
		t.Fatalf("expected .pcm suffix, got %s", path)
	}
	data := []byte{1, 2, 3, 4}
	if err := store.WriteAll(path, data); err != nil {
		t.Fatalf("write all: %v", err)
	}
	got, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(got) != len(data) {
		t.Fatalf("unexpected len: got %d want %d", len(got), len(data))
	}
}
