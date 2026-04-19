//go:build integration

package integration_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"git.produktor.io/edelweiss/docs/services/internal/config"
	"git.produktor.io/edelweiss/docs/services/internal/docsparse"
	"git.produktor.io/edelweiss/docs/services/internal/embed"
	"git.produktor.io/edelweiss/docs/services/internal/graph"
	"git.produktor.io/edelweiss/docs/services/internal/llm"
	"git.produktor.io/edelweiss/docs/services/internal/rag"
	"git.produktor.io/edelweiss/docs/services/internal/vectorstore"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/neo4j"
	"github.com/testcontainers/testcontainers-go/modules/qdrant"
)

// TestRAGFunctional runs ingest + query against real Ollama when RUN_RAG_FUNCTIONAL=1.
func TestRAGFunctional(t *testing.T) {
	if os.Getenv("RUN_RAG_FUNCTIONAL") != "1" {
		t.Skip("set RUN_RAG_FUNCTIONAL=1 to run (needs docker + Ollama + time)")
	}
	ctx := context.Background()

	ncont, err := neo4j.Run(ctx, "neo4j:5-community", neo4j.WithAdminPassword("rag-func-test"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = ncont.Terminate(ctx) })
	bolt, err := ncont.BoltUrl(ctx)
	require.NoError(t, err)

	qc, err := qdrant.Run(ctx, "qdrant/qdrant:latest")
	require.NoError(t, err)
	t.Cleanup(func() { _ = qc.Terminate(ctx) })
	qURL, err := qc.RESTEndpoint(ctx)
	require.NoError(t, err)

	t.Setenv("NEO4J_URI", bolt)
	t.Setenv("NEO4J_USER", "neo4j")
	t.Setenv("NEO4J_PASSWORD", "rag-func-test")
	t.Setenv("QDRANT_URL", qURL)
	t.Setenv("QDRANT_COLLECTION", "edelweiss_func_test")
	t.Setenv("OLLAMA_URL", ollamaURL())
	t.Setenv("DOCS_ROOT", docsProjectDir(t))

	cfg, err := config.Load()
	require.NoError(t, err)

	gstore, err := graph.NewStore(ctx, cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword)
	require.NoError(t, err)
	t.Cleanup(func() { _ = gstore.Close(ctx) })

	pr, err := docsparse.WalkDocs(cfg.DocsRoot)
	require.NoError(t, err)
	require.NoError(t, gstore.WriteCorpus(ctx, pr))

	emb := embed.New(cfg.OllamaURL, cfg.HTTPTimeout)
	probe, err := emb.Embed(ctx, cfg.EmbedModel, "probe")
	if err != nil {
		t.Skipf("ollama: %v", err)
	}
	dim := uint64(len(probe))
	q := vectorstore.New(cfg.QdrantURL, cfg.HTTPTimeout)
	require.NoError(t, q.EnsureCollection(ctx, cfg.QdrantCollection, dim))

	for _, ch := range pr.Chunks {
		vec, err := emb.Embed(ctx, cfg.EmbedModel, ch.Text)
		require.NoError(t, err)
		h := docsparse.ChunkContentHash(ch)
		pt := vectorstore.PointFromChunk(h, vec, map[string]any{
			"node_id": ch.NodeID,
			"path":    ch.Path,
			"text":    ch.Text,
		})
		require.NoError(t, q.UpsertPoints(ctx, cfg.QdrantCollection, []vectorstore.Point{pt}))
	}

	llmClient := llm.New(cfg.OllamaURL, cfg.HTTPTimeout)
	engine := rag.BuildEngineFromConfig(emb, llmClient, q, gstore, cfg.EmbedModel, cfg.GenModel, cfg.QdrantCollection)

	ans, err := engine.Answer(ctx, "Как устроена помесячная Abrechnung (UC-07)?")
	require.NoError(t, err)
	require.True(t, strings.Contains(strings.ToUpper(ans), "UC-07") || strings.Contains(ans, "Abrechnung"))
}
