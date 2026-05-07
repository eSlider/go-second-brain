package assistant

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type MediaStore struct {
	baseDir string
}

func NewMediaStore(baseDir string) (*MediaStore, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("assistant storage: baseDir is required")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("assistant storage: mkdir %s: %w", baseDir, err)
	}
	return &MediaStore{baseDir: baseDir}, nil
}

func (s *MediaStore) NewFilePath(format string) string {
	id := randomID()
	ts := time.Now().UTC().Format("20060102T150405.000000000Z")
	return filepath.Join(s.baseDir, fmt.Sprintf("%s-%s.%s", id, ts, format))
}

func (s *MediaStore) WriteAll(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("assistant storage: write %s: %w", path, err)
	}
	return nil
}

func randomID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
