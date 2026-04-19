// Command ingestor walks docs/project, updates Neo4j and Qdrant.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.produktor.io/edelweiss/docs/services/internal/config"
	"git.produktor.io/edelweiss/docs/services/internal/docsparse"
	"git.produktor.io/edelweiss/docs/services/internal/embed"
	"git.produktor.io/edelweiss/docs/services/internal/graph"
	"git.produktor.io/edelweiss/docs/services/internal/slogx"
	"git.produktor.io/edelweiss/docs/services/internal/vectorstore"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.Any("err", err))
		return 1
	}
	log := slogx.New(cfg.MatrixDebug)
	runStart := time.Now()

	parseStart := time.Now()
	pr, err := docsparse.WalkDocs(cfg.DocsRoot)
	if err != nil {
		log.Error("walk docs", slog.Any("err", err), slog.String("root", cfg.DocsRoot))
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

	gstore, err := graph.NewStore(ctx, cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword)
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

	emb := embed.New(cfg.OllamaURL, cfg.HTTPTimeout)
	probe, err := emb.Embed(ctx, cfg.EmbedModel, "dimension probe")
	if err != nil {
		log.Error("embed probe", slog.Any("err", err))
		return 1
	}
	dim := uint64(len(probe))
	q := vectorstore.New(cfg.QdrantURL, cfg.HTTPTimeout)
	if err := q.EnsureCollection(ctx, cfg.QdrantCollection, dim); err != nil {
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
		var points []vectorstore.Point
		for _, ch := range pr.Chunks[i:end] {
			h := docsparse.ChunkContentHash(ch)
			vec, err := emb.Embed(ctx, cfg.EmbedModel, ch.Text)
			if err != nil {
				log.Error("embed chunk", slog.String("path", ch.Path), slog.Any("err", err))
				return 1
			}
			if uint64(len(vec)) != dim {
				log.Error("embedding dim mismatch", slog.Int("got", len(vec)), slog.Uint64("want", dim))
				return 1
			}
			points = append(points, vectorstore.PointFromChunk(h, vec, map[string]any{
				"node_id":      ch.NodeID,
				"path":         ch.Path,
				"heading":      ch.Heading,
				"text":         ch.Text,
				"content_hash": h,
			}))
		}
		if err := q.UpsertPoints(ctx, cfg.QdrantCollection, points); err != nil {
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
