package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"git.produktor.io/edelweiss/docs/services/internal/assistant"
	"git.produktor.io/edelweiss/docs/services/internal/config"
	"git.produktor.io/edelweiss/docs/services/internal/slogx"
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
	if err := cfg.Assistant(); err != nil {
		slog.Error("assistant config", slog.Any("err", err))
		return 1
	}
	log := slogx.New(cfg.MatrixDebug)
	perf, err := assistant.NewPerfLogger(cfg.AssistantPerfLog)
	if err != nil {
		log.Error("perf logger", slog.Any("err", err))
		return 1
	}
	defer func() {
		_ = perf.Close()
	}()

	rt := assistant.NewRuntime(cfg, log, perf)
	log.Info("assistant started")
	if err := rt.Run(ctx); err != nil && ctx.Err() == nil {
		log.Error("assistant run", slog.Any("err", err))
		return 1
	}
	log.Info("assistant stopped")
	return 0
}
