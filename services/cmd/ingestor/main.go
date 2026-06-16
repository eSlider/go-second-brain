// Command ingestor walks DOCS_ROOT (Markdown corpus), updates Neo4j and Qdrant.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eSlider/go-second-brain/services/internal/config"
	"github.com/eSlider/go-second-brain/services/internal/docsparse"
	"github.com/eSlider/go-second-brain/services/internal/graph"
	"github.com/eSlider/go-second-brain/services/internal/slogx"
	"github.com/eSlider/go-second-brain/services/pkg/ollama"
	"github.com/eSlider/go-second-brain/services/pkg/qdrant"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.NewConfig()
	if err := cfg.Load(); err != nil {
		slog.Error("config", slog.Any("err", err))
		return 1
	}
	log := slogx.New(cfg.Matrix.Debug)
	runStart := time.Now()

	parseStart := time.Now()
	pr, err := docsparse.WalkDocs(cfg.Docs.Root)
	if err != nil {
		log.Error("walk docs", slog.Any("err", err), slog.String("root", cfg.Docs.Root))
		return 1
	}
	log.Info("ingest stage",
		slog.String("event", "ingest_parsed"),
		slog.String("service", "kg-ingestor"),
		slog.Int("nodes", len(pr.Nodes)),
		slog.Int("edges", len(pr.Edges)),
		slog.Int("chunks", len(pr.Chunks)),
		slog.Int64("latency_ms", time.Since(parseStart).Milliseconds()),
	)

	gstore, err := graph.NewStore(ctx, cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		log.Error("neo4j", slog.Any("err", err))
		return 1
	}
	defer func() {
		_ = gstore.Close(ctx)
	}()
	graphStart := time.Now()
	if err := gstore.WriteCorpus(ctx, pr); err != nil {
		log.Error("write graph", slog.Any("err", err))
		return 1
	}
	log.Info("ingest stage",
		slog.String("event", "ingest_graph_written"),
		slog.String("service", "kg-ingestor"),
		slog.Int("nodes", len(pr.Nodes)),
		slog.Int("edges", len(pr.Edges)),
		slog.Int64("latency_ms", time.Since(graphStart).Milliseconds()),
	)

	llmCli, err := ollama.New(ctx, &ollama.Config{URL: cfg.Ollama.URL, Timeout: cfg.HTTP.Timeout})
	if err != nil {
		log.Error("ollama", slog.Any("err", err))
		return 1
	}
	defer func() {
		_ = llmCli.Close()
	}()
	q, err := qdrant.New(ctx, &cfg.Qdrant, cfg.HTTP.Timeout)
	if err != nil {
		log.Error("qdrant", slog.Any("err", err))
		return 1
	}
	defer func() {
		_ = q.Close()
	}()
	probe, err := llmCli.Embed(ctx, cfg.Embedding.Model, "dimension probe")
	if err != nil {
		log.Error("embed probe", slog.Any("err", err))
		return 1
	}
	dim := uint64(len(probe))
	if err := q.EnsureCollection(ctx, cfg.Qdrant.Collection, dim); err != nil {
		log.Error("qdrant collection", slog.Any("err", err))
		return 1
	}

	const batch = 64
	embedStart := time.Now()
	for i := 0; i < len(pr.Chunks); i += batch {
		end := i + batch
		if end > len(pr.Chunks) {
			end = len(pr.Chunks)
		}
		batchStart := time.Now()
		var points []qdrant.Point
		for _, ch := range pr.Chunks[i:end] {
			h := docsparse.ChunkContentHash(ch)
			vec, err := llmCli.Embed(ctx, cfg.Embedding.Model, ch.Text)
			if err != nil {
				log.Error("embed chunk", slog.String("path", ch.Path), slog.Any("err", err))
				return 1
			}
			if uint64(len(vec)) != dim {
				log.Error("embedding dim mismatch", slog.Int("got", len(vec)), slog.Uint64("want", dim))
				return 1
			}
			points = append(points, qdrant.PointFromChunk(h, vec, map[string]any{
				"node_id":      ch.NodeID,
				"path":         ch.Path,
				"heading":      ch.Heading,
				"text":         ch.Text,
				"content_hash": h,
			}))
		}
		if err := q.UpsertPoints(ctx, cfg.Qdrant.Collection, points); err != nil {
			log.Error("qdrant upsert", slog.Any("err", err))
			return 1
		}
		log.Info("ingest stage",
			slog.String("event", "ingest_batch_upserted"),
			slog.String("service", "kg-ingestor"),
			slog.Int("from", i),
			slog.Int("to", end),
			slog.Int("count", end-i),
			slog.Int64("latency_ms", time.Since(batchStart).Milliseconds()),
		)
	}
	log.Info("ingest stage",
		slog.String("event", "ingest_complete"),
		slog.String("service", "kg-ingestor"),
		slog.Int("nodes", len(pr.Nodes)),
		slog.Int("edges", len(pr.Edges)),
		slog.Int("chunks", len(pr.Chunks)),
		slog.Int64("embed_latency_ms", time.Since(embedStart).Milliseconds()),
		slog.Int64("latency_ms", time.Since(runStart).Milliseconds()),
	)
	return 0
}
