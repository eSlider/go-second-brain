// Package vectorstore implements Qdrant REST operations used by ingestor and RAG.
package vectorstore

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"git.produktor.io/edelweiss/docs/services/internal/httpjson"
)

// Qdrant wraps HTTP calls to Qdrant.
type Qdrant struct {
	HTTP *httpjson.Client
}

// New creates a client (baseURL e.g. http://localhost:6333).
func New(baseURL string, timeout time.Duration) *Qdrant {
	return &Qdrant{HTTP: httpjson.New(baseURL, timeout)}
}

type createCollectionBody struct {
	Vectors vectorsConfig `json:"vectors"`
}

type vectorsConfig struct {
	Size     uint64 `json:"size"`
	Distance string `json:"distance"`
}

// EnsureCollection creates the collection if it does not exist.
func (q *Qdrant) EnsureCollection(ctx context.Context, name string, vectorSize uint64) error {
	path := "/collections/" + strings.Trim(name, "/")
	var got map[string]any
	if err := q.HTTP.GetJSON(ctx, path, &got); err == nil {
		return nil
	}
	errPut := q.HTTP.PutJSON(ctx, path, createCollectionBody{
		Vectors: vectorsConfig{Size: vectorSize, Distance: "Cosine"},
	}, &map[string]any{})
	if errPut == nil {
		return nil
	}
	if err2 := q.HTTP.GetJSON(ctx, path, &got); err2 == nil {
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
func (q *Qdrant) UpsertPoints(ctx context.Context, collection string, points []Point) error {
	if len(points) == 0 {
		return nil
	}
	path := fmt.Sprintf("/collections/%s/points?wait=true", strings.Trim(collection, "/"))
	return q.HTTP.PutJSON(ctx, path, upsertBody{Points: points}, &map[string]any{})
}

// PointFromChunk builds a Qdrant point with deterministic ID from chunkID string.
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
func (q *Qdrant) Search(ctx context.Context, collection string, vector []float32, limit int) ([]SearchHit, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 256 {
		limit = 256
	}
	// limit is in [1,256] after checks; safe for uint64 on 64-bit arch.
	lim := uint64(limit) //nolint:gosec // G115
	path := fmt.Sprintf("/collections/%s/points/search", strings.Trim(collection, "/"))
	var out searchResponse
	if err := q.HTTP.PostJSON(ctx, path, searchBody{
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
