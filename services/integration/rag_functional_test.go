//go:build integration

package integration_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"git.produktor.io/edelweiss/docs/services/internal/config"
	"git.produktor.io/edelweiss/docs/services/internal/docsparse"
	"git.produktor.io/edelweiss/docs/services/internal/graph"
	"git.produktor.io/edelweiss/docs/services/internal/rag"
	"git.produktor.io/edelweiss/docs/services/pkg/ollama"
	qdrantpkg "git.produktor.io/edelweiss/docs/services/pkg/qdrant"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/neo4j"
	tcqdrant "github.com/testcontainers/testcontainers-go/modules/qdrant"
)

const defaultGenModelRAGTest = "cajina/gemma4_e2b-q4_k_s:v01"

// TestRAGFunctional runs full ingest + RAG against real Ollama when RUN_RAG_FUNCTIONAL=1.
// No Matrix/Synapse: only Neo4j + Qdrant + Ollama (dialogue subtest simulates multi-turn Q&A).
func TestRAGFunctional(t *testing.T) {
	//if os.Getenv("RUN_RAG_FUNCTIONAL") != "1" {
	//	t.Skip("set RUN_RAG_FUNCTIONAL=1 to run (needs docker + Ollama + time)")
	//}
	ctx := context.Background()

	t.Setenv("HTTP_TIMEOUT", "600s")
	if os.Getenv("GEN_MODEL") == "" {
		t.Setenv("GEN_MODEL", defaultGenModelRAGTest)
	}

	ncont, err := neo4j.Run(ctx, "neo4j:5-community", neo4j.WithAdminPassword("rag-func-test"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = ncont.Terminate(ctx) })
	bolt, err := ncont.BoltUrl(ctx)
	require.NoError(t, err)

	qc, err := tcqdrant.Run(ctx, "qdrant/qdrant:latest")
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

	cfg := config.NewConfig()
	require.NoError(t, cfg.Load())

	gstore, err := graph.NewStore(ctx, cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	require.NoError(t, err)
	t.Cleanup(func() { _ = gstore.Close(ctx) })

	pr, err := docsparse.WalkDocs(cfg.Docs.Root)
	require.NoError(t, err)
	require.NoError(t, gstore.WriteCorpus(ctx, pr))

	llmCli, err := ollama.New(ctx, &ollama.Config{URL: cfg.Ollama.URL, Timeout: cfg.HTTP.Timeout})
	require.NoError(t, err)
	t.Cleanup(func() { _ = llmCli.Close() })
	q, err := qdrantpkg.New(ctx, &cfg.Qdrant, cfg.HTTP.Timeout)
	require.NoError(t, err)
	t.Cleanup(func() { _ = q.Close() })
	probe, err := llmCli.Embed(ctx, cfg.Embedding.Model, "probe")
	if err != nil {
		t.Skipf("ollama embed: %v", err)
	}
	dim := uint64(len(probe))
	require.NoError(t, q.EnsureCollection(ctx, cfg.Qdrant.Collection, dim))

	for _, ch := range pr.Chunks {
		vec, err := llmCli.Embed(ctx, cfg.Embedding.Model, ch.Text)
		require.NoError(t, err)
		h := docsparse.ChunkContentHash(ch)
		pt := qdrantpkg.PointFromChunk(h, vec, map[string]any{
			"node_id": ch.NodeID,
			"path":    ch.Path,
			"text":    ch.Text,
		})
		require.NoError(t, q.UpsertPoints(ctx, cfg.Qdrant.Collection, []qdrantpkg.Point{pt}))
	}

	engine := rag.BuildEngineFromConfig(llmCli, q, gstore, cfg.Embedding.Model, cfg.Generator.Model, cfg.Qdrant.Collection)

	t.Run("single_query_uc07", func(t *testing.T) {
		cctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
		ans, err := engine.Answer(cctx, "Как устроена помесячная Abrechnung (UC-07)?")
		require.NoError(t, err)
		require.True(t, strings.Contains(strings.ToUpper(ans), "UC-07") || strings.Contains(ans, "Abrechnung"))
	})

	t.Run("dialogue_multiturn_without_matrix", func(t *testing.T) {
		// Sequential RAG turns (no Synapse): each question is independent; simulates a chat check.
		turns := []struct {
			name  string
			query string
			min   int
		}{
			{"turn1_uc01", "Кратко: что описывает кейс UC-01?", 80},
			{"turn2_subjects", "Какие роли SUBJ упоминаются рядом с intake в базе?", 60},
			{"turn3_term", "Что такое Pflegegrad одним предложением по глоссарию?", 40},
		}
		for _, step := range turns {
			step := step
			t.Run(step.name, func(t *testing.T) {
				cctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
				defer cancel()
				ans, err := engine.Answer(cctx, step.query)
				require.NoError(t, err, step.name)
				trim := strings.TrimSpace(ans)
				require.GreaterOrEqual(t, len(trim), step.min, "answer too short: %q", truncateForLog(trim, 200))
			})
		}
	})
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
