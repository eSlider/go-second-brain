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

	"git.produktor.io/edelweiss/docs/services/internal/config"
	"git.produktor.io/edelweiss/docs/services/internal/embed"
	"git.produktor.io/edelweiss/docs/services/internal/graph"
	"git.produktor.io/edelweiss/docs/services/internal/llm"
	"git.produktor.io/edelweiss/docs/services/internal/rag"
	"git.produktor.io/edelweiss/docs/services/internal/slogx"
	"git.produktor.io/edelweiss/docs/services/internal/vectorstore"
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

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.Any("err", err))
		return 1
	}
	if err := cfg.Bot(); err != nil {
		slog.Error("bot config", slog.Any("err", err))
		return 1
	}
	log := slogx.New(cfg.MatrixDebug)

	gstore, err := graph.NewStore(ctx, cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword)
	if err != nil {
		log.Error("neo4j", slog.Any("err", err))
		return 1
	}
	defer func() {
		_ = gstore.Close(ctx)
	}()

	emb := embed.New(cfg.OllamaURL, cfg.HTTPTimeout)
	llmClient := llm.New(cfg.OllamaURL, cfg.HTTPTimeout)
	qdr := vectorstore.New(cfg.QdrantURL, cfg.HTTPTimeout)
	probe, err := emb.Embed(ctx, cfg.EmbedModel, "dimension probe")
	if err != nil {
		log.Error("ollama embed probe", slog.Any("err", err), slog.String("model", cfg.EmbedModel))
		return 1
	}
	dim := uint64(len(probe))
	if err := qdr.EnsureCollection(ctx, cfg.QdrantCollection, dim); err != nil {
		log.Error("qdrant ensure collection", slog.Any("err", err), slog.String("collection", cfg.QdrantCollection))
		return 1
	}
	log.Info("rag backend ready", slog.String("collection", cfg.QdrantCollection), slog.Int("embed_dim", len(probe)))

	engine := rag.BuildEngineFromConfig(emb, llmClient, qdr, gstore, cfg.EmbedModel, cfg.GenModel, cfg.QdrantCollection)

	bcfg := matrix.Config{
		Homeserver: cfg.MatrixHomeserver,
		Username:   cfg.MatrixUser,
		Password:   cfg.MatrixPassword,
		Database:   cfg.BotDBPath,
		Debug:      cfg.MatrixDebug,
	}
	bot, err := matrix.NewBot(bcfg)
	if err != nil {
		log.Error("matrix bot", slog.Any("err", err))
		return 1
	}

	prefix := strings.TrimSpace(cfg.BotCommandPrefix)
	if prefix == "" {
		prefix = "!edel"
	}

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
		ans, err := engine.Answer(c, query)
		if err != nil {
			log.Error("rag", slog.Any("err", err))
			_ = bot.SendText(c, roomID, "Ошибка при генерации ответа. Проверьте Ollama и индекс.")
			return
		}
		md := ans
		html := matrix.MarkdownToHTML(md)
		if err := bot.SendReply(c, roomID, md, html, sender); err != nil {
			log.Error("send reply", slog.Any("err", err))
		}
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
