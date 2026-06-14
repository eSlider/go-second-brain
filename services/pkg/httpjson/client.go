// Package httpjson provides a small HTTP JSON helper for outbound API calls.
package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client wraps http.Client with JSON helpers.
type Client struct {
	HTTP    *http.Client
	BaseURL string
}

// New returns a Client with explicit timeout (never zero).
func New(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP: &http.Client{
			Timeout: timeout,
		},
	}
}

// PostJSON posts JSON body and decodes JSON response into out.
func (c *Client) PostJSON(ctx context.Context, path string, body any, out any) error {
	u := c.BaseURL + path
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("httpjson: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("httpjson: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("httpjson: do: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("httpjson: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("httpjson: %s: %s", resp.Status, truncate(string(raw), 512))
	}
	if out == nil {
		return nil
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("httpjson: decode: %w", err)
	}
	return nil
}

// PutJSON sends PUT with JSON body.
func (c *Client) PutJSON(ctx context.Context, path string, body any, out any) error {
	u := c.BaseURL + path
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("httpjson: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("httpjson: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("httpjson: do: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("httpjson: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("httpjson: %s: %s", resp.Status, truncate(string(raw), 512))
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("httpjson: decode: %w", err)
	}
	return nil
}

// PostRaw posts JSON and returns the raw response body.
func (c *Client) PostRaw(ctx context.Context, path string, body any) ([]byte, error) {
	u := c.BaseURL + path
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("httpjson: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("httpjson: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpjson: do: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("httpjson: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("httpjson: %s: %s", resp.Status, truncate(string(raw), 512))
	}
	return raw, nil
}

// GetJSON GETs and decodes JSON.
func (c *Client) GetJSON(ctx context.Context, path string, out any) error {
	u := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("httpjson: request: %w", err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("httpjson: do: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("httpjson: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("httpjson: %s: %s", resp.Status, truncate(string(raw), 512))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("httpjson: decode: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
