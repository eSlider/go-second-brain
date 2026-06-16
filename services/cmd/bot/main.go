// Command bot connects to Matrix and answers with RAG over the knowledge base.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/eSlider/go-second-brain/services/internal/config"
	"github.com/eSlider/go-second-brain/services/internal/graph"
	"github.com/eSlider/go-second-brain/services/internal/rag"
	"github.com/eSlider/go-second-brain/services/internal/slogx"
	"github.com/eSlider/go-second-brain/services/pkg/ollama"
	"github.com/eSlider/go-second-brain/services/pkg/qdrant"
	matrix "github.com/eslider/go-matrix-bot"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
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
	if err := cfg.Bot(); err != nil {
		slog.Error("bot config", slog.Any("err", err))
		return 1
	}
	log := slogx.New(cfg.Matrix.Debug)

	gstore, err := graph.NewStore(ctx, cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		log.Error("neo4j", slog.Any("err", err))
		return 1
	}
	defer func() {
		_ = gstore.Close(ctx)
	}()

	llmCli, err := ollama.New(ctx, &ollama.Config{URL: cfg.Ollama.URL, Timeout: cfg.HTTP.Timeout})
	if err != nil {
		log.Error("ollama", slog.Any("err", err))
		return 1
	}
	defer func() {
		_ = llmCli.Close()
	}()
	qdr, err := qdrant.New(ctx, &cfg.Qdrant, cfg.HTTP.Timeout)
	if err != nil {
		log.Error("qdrant client", slog.Any("err", err))
		return 1
	}
	defer func() {
		_ = qdr.Close()
	}()
	dim, err := waitForRAGBackend(ctx, log, llmCli, qdr, cfg.Embedding.Model, cfg.Qdrant.Collection)
	if err != nil {
		log.Error("rag backend", slog.Any("err", err))
		return 1
	}
	log.Info("rag backend ready", slog.String("collection", cfg.Qdrant.Collection), slog.Int("embed_dim", dim))

	engine := rag.BuildEngineFromConfig(llmCli, qdr, cfg.Embedding.Model, cfg.Generator.Model, cfg.Qdrant.Collection, cfg.RAG.TopK, cfg.RAG.SystemPrompt)
	bot, err := matrix.NewBot(matrix.Config{
		Homeserver: cfg.Matrix.Homeserver(),
		Username:   cfg.Matrix.ResolvedUser(),
		Password:   cfg.Matrix.ResolvedPassword(),
		Database:   cfg.Matrix.Bot.DB,
		Debug:      cfg.Matrix.Debug,
	})
	if err != nil {
		log.Error("matrix bot", slog.Any("err", err))
		return 1
	}

	prefix := cfg.Commands.Command.Prefix

	bot.OnMessage(func(c context.Context, roomID id.RoomID, sender id.UserID, msg *event.MessageEventContent) {
		if msg == nil {
			return
		}
		cli := bot.Client()
		if cli == nil {
			return
		}
		if sender == cli.UserID {
			return
		}
		body := strings.TrimSpace(msg.Body)
		if body == "" {
			return
		}
		query := queryFromMessage(body, prefix)
		started := time.Now()
		ans, err := engine.Answer(c, query)
		latency := time.Since(started)

		base := []any{
			slog.String("event", "bot_query"),
			slog.String("service", "matrix-bot"),
			slog.String("room_id", string(roomID)),
			slog.String("sender", string(sender)),
			slog.Int("query_len", len(query)),
			slog.Int64("latency_ms", latency.Milliseconds()),
		}
		if err != nil {
			log.Error("bot query failed", append(base,
				slog.Bool("ok", false),
				slog.String("err", err.Error()),
			)...)
			_ = bot.SendText(c, roomID, "Ошибка при генерации ответа. Проверьте Ollama и индекс.")
			return
		}
		md := ans
		html := matrix.MarkdownToHTML(md)
		sendErr := bot.SendReply(c, roomID, md, html, sender)
		fields := append(base,
			slog.Bool("ok", sendErr == nil),
			slog.Int("answer_len", len(ans)),
		)
		if sendErr != nil {
			log.Error("bot reply failed", append(fields, slog.String("err", sendErr.Error()))...)
			return
		}
		log.Info("bot query answered", fields...)
	})

	if err := bot.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("bot run", slog.Any("err", err))
		return 1
	}
	if err := bot.Stop(); err != nil {
		log.Error("bot stop", slog.Any("err", err))
		return 1
	}
	return 0
}

// waitForRAGBackend blocks until Ollama embeddings and Qdrant collection are reachable.
// Retries forever with exponential backoff (capped) so the container does not exit-restart
// while Neo4j/Qdrant/Ollama start up.
func waitForRAGBackend(
	ctx context.Context,
	log *slog.Logger,
	emb *ollama.Client,
	qdr *qdrant.Client,
	embedModel, collection string,
) (int, error) {
	backoff := 2 * time.Second
	const maxBackoff = 60 * time.Second
	attempt := 0
	for {
		attempt++
		probe, err := emb.Embed(ctx, embedModel, "dimension probe")
		if err != nil {
			log.Warn("ollama not ready, retrying",
				slog.Any("err", err), slog.String("model", embedModel),
				slog.Int("attempt", attempt), slog.Duration("next_in", backoff))
			if err := sleepCtx(ctx, backoff); err != nil {
				return 0, err
			}
			if backoff < maxBackoff {
				nb := backoff * 2
				if nb > maxBackoff {
					nb = maxBackoff
				}
				backoff = nb
			}
			continue
		}
		dim := len(probe)
		if err := qdr.EnsureCollection(ctx, collection, uint64(dim)); err != nil {
			log.Warn("qdrant not ready, retrying",
				slog.Any("err", err), slog.String("collection", collection),
				slog.Int("attempt", attempt), slog.Duration("next_in", backoff))
			if err := sleepCtx(ctx, backoff); err != nil {
				return 0, err
			}
			continue
		}
		return dim, nil
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// queryFromMessage uses the full message as the RAG query; if it starts with the command prefix
// (case-insensitive), that prefix is stripped.
func queryFromMessage(body, prefix string) string {
	b := strings.TrimSpace(body)
	p := strings.TrimSpace(prefix)
	if p != "" && len(b) >= len(p) && strings.EqualFold(b[:len(p)], p) {
		q := strings.TrimSpace(b[len(p):])
		if q != "" {
			return q
		}
	}
	return b
}
