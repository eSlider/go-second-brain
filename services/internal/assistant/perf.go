package assistant

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PerfLogger struct {
	mu   sync.Mutex
	f    *os.File
	sid  string
	base time.Time
}

func NewPerfLogger(path string) (*PerfLogger, error) {
	if path == "" {
		return nil, fmt.Errorf("assistant perf: path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("assistant perf: mkdir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("assistant perf: open: %w", err)
	}
	return &PerfLogger{
		f:    f,
		sid:  randomID(),
		base: time.Now(),
	}, nil
}

func (p *PerfLogger) Event(name string, fields map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	payload := map[string]any{
		"ts":         time.Now().UTC().Format(time.RFC3339Nano),
		"session_id": p.sid,
		"event":      name,
		"since_ms":   time.Since(p.base).Milliseconds(),
	}
	for k, v := range fields {
		payload[k] = v
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("assistant perf: marshal: %w", err)
	}
	if _, err := p.f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("assistant perf: write: %w", err)
	}
	return nil
}

func (p *PerfLogger) Close() error {
	if p == nil || p.f == nil {
		return nil
	}
	return p.f.Close()
}
