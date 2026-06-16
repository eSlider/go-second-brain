//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eSlider/go-second-brain/services/internal/docsparse"
	"github.com/eSlider/go-second-brain/services/internal/graph"
	"github.com/eSlider/go-second-brain/services/pkg/ollama"
	qdrantpkg "github.com/eSlider/go-second-brain/services/pkg/qdrant"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/neo4j"
	tcqdrant "github.com/testcontainers/testcontainers-go/modules/qdrant"
)

func ollamaURL() string {
	u := os.Getenv("OLLAMA_URL")
	if u == "" {
		u = "http://127.0.0.1:11434"
	}
	return u
}

func corpusDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "examples", "corpus"))
	require.NoError(t, err)
	return root
}

func TestNeo4jWriteCorpus(t *testing.T) {
	ctx := context.Background()
	ncont, err := neo4j.Run(ctx, "neo4j:5-community", neo4j.WithAdminPassword("integration-test"))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = ncont.Terminate(ctx)
	})
	bolt, err := ncont.BoltUrl(ctx)
	require.NoError(t, err)

	st, err := graph.NewStore(ctx, bolt, "neo4j", "integration-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = st.Close(ctx)
	})

	pr, err := docsparse.WalkDocs(corpusDir(t))
	require.NoError(t, err)
	require.NotEmpty(t, pr.Nodes)

	err = st.WriteCorpus(ctx, pr)
	require.NoError(t, err)
}

func TestQdrantEmbedAndSearch(t *testing.T) {
	ctx := context.Background()
	qc, err := tcqdrant.Run(ctx, "qdrant/qdrant:latest")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = qc.Terminate(ctx)
	})
	httpURL, err := qc.RESTEndpoint(ctx)
	require.NoError(t, err)

	embCli, err := ollama.New(ctx, &ollama.Config{URL: ollamaURL(), Timeout: 60 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { _ = embCli.Close() })
	_, err = embCli.Embed(ctx, getenvDefault("EMBED_MODEL", "embeddinggemma"), "ping")
	if err != nil {
		t.Skipf("ollama not reachable: %v", err)
	}

	vec, err := embCli.Embed(ctx, getenvDefault("EMBED_MODEL", "embeddinggemma"), "sample workflow knowledge node")
	require.NoError(t, err)
	dim := uint64(len(vec))

	q, err := qdrantpkg.New(ctx, &qdrantpkg.Config{URL: httpURL}, 30*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = q.Close() })
	name := "test_knowledge_integration"
	require.NoError(t, q.EnsureCollection(ctx, name, dim))

	ch := docsparse.TextChunk{
		NodeID:  "UC-02",
		Heading: "sample process",
		Path:    "cases/UC-02-sample-process.md",
		Text:    "UC-02 sample process monthly review SUBJ-REVIEWER",
	}
	h := docsparse.ChunkContentHash(ch)
	vec2, err := embCli.Embed(ctx, getenvDefault("EMBED_MODEL", "embeddinggemma"), ch.Text)
	require.NoError(t, err)

	pt := qdrantpkg.PointFromChunk(h, vec2, map[string]any{
		"node_id": ch.NodeID,
		"path":    ch.Path,
		"text":    ch.Text,
	})
	require.NoError(t, q.UpsertPoints(ctx, name, []qdrantpkg.Point{pt}))

	qvec, err := embCli.Embed(ctx, getenvDefault("EMBED_MODEL", "embeddinggemma"), "monthly review process")
	require.NoError(t, err)
	hits, err := q.Search(ctx, name, qvec, 3)
	require.NoError(t, err)
	require.NotEmpty(t, hits)
}

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
