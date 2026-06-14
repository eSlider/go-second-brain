package qdrant

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"git.produktor.io/edelweiss/docs/services/pkg/httpjson"
)

// Client wraps HTTP calls to Qdrant.
type Client struct {
	HTTP *httpjson.Client
}

// New constructs a Qdrant client from cfg.URL and opts.Timeout override.
func New(_ context.Context, cfg *Config, timeout time.Duration) (*Client, error) {
	if cfg == nil || strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("qdrant: URL is required")
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Client{HTTP: httpjson.New(cfg.URL, timeout)}, nil
}

// Close is a lifecycle hook for parity with other clients (no pooled resources yet).
func (c *Client) Close() error { return nil }

type createCollectionBody struct {
	Vectors vectorsConfig `json:"vectors"`
}

type vectorsConfig struct {
	Size     uint64 `json:"size"`
	Distance string `json:"distance"`
}

// EnsureCollection creates the collection if it does not exist.
func (c *Client) EnsureCollection(ctx context.Context, name string, vectorSize uint64) error {
	path := "/collections/" + strings.Trim(name, "/")
	var got map[string]any
	if err := c.HTTP.GetJSON(ctx, path, &got); err == nil {
		return nil
	}
	errPut := c.HTTP.PutJSON(ctx, path, createCollectionBody{
		Vectors: vectorsConfig{Size: vectorSize, Distance: "Cosine"},
	}, &map[string]any{})
	if errPut == nil {
		return nil
	}
	if err2 := c.HTTP.GetJSON(ctx, path, &got); err2 == nil {
		return nil
	}
	return errPut
}

type upsertBody struct {
	Points []Point `json:"points"`
}

// Point is a single vector row in Qdrant.
type Point struct {
	ID      uint64         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

// UpsertPoints writes or updates vectors (wait=true).
func (c *Client) UpsertPoints(ctx context.Context, collection string, points []Point) error {
	if len(points) == 0 {
		return nil
	}
	path := fmt.Sprintf("/collections/%s/points?wait=true", strings.Trim(collection, "/"))
	return c.HTTP.PutJSON(ctx, path, upsertBody{Points: points}, &map[string]any{})
}

// PointFromChunk builds a point with deterministic ID from chunkID string.
func PointFromChunk(chunkID string, vector []float32, payload map[string]any) Point {
	return Point{
		ID:      deterministicPointID(chunkID),
		Vector:  vector,
		Payload: payload,
	}
}

func deterministicPointID(chunkID string) uint64 {
	h := sha256.Sum256([]byte(chunkID))
	return binary.BigEndian.Uint64(h[:8])
}

// SearchHit is one search result.
type SearchHit struct {
	ID      uint64         `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
}

type searchBody struct {
	Vector      []float32 `json:"vector"`
	Limit       uint64    `json:"limit"`
	WithPayload bool      `json:"with_payload"`
	WithVectors bool      `json:"with_vector"`
}

type searchResponse struct {
	Result []struct {
		ID      uint64         `json:"id"`
		Version uint64         `json:"version"`
		Score   float64        `json:"score"`
		Payload map[string]any `json:"payload"`
	} `json:"result"`
}

// Search returns nearest neighbors.
func (c *Client) Search(ctx context.Context, collection string, vector []float32, limit int) ([]SearchHit, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 256 {
		limit = 256
	}
	lim := uint64(limit) //nolint:gosec // G115 — clamped to [1,256]
	path := fmt.Sprintf("/collections/%s/points/search", strings.Trim(collection, "/"))
	var out searchResponse
	if err := c.HTTP.PostJSON(ctx, path, searchBody{
		Vector:      vector,
		Limit:       lim,
		WithPayload: true,
		WithVectors: false,
	}, &out); err != nil {
		return nil, err
	}
	res := make([]SearchHit, 0, len(out.Result))
	for _, r := range out.Result {
		res = append(res, SearchHit{ID: r.ID, Score: r.Score, Payload: r.Payload})
	}
	return res, nil
}

// HexID exposes deterministic id as hex (for logs).
func HexID(chunkID string) string {
	h := sha256.Sum256([]byte(chunkID))
	return hex.EncodeToString(h[:8])
}
