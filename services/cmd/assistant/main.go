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

	cfg := config.NewConfig()
	if err := cfg.Load(); err != nil {
		slog.Error("config", slog.Any("err", err))
		return 1
	}
	if err := cfg.ValidateAssistant(); err != nil {
		slog.Error("assistant config", slog.Any("err", err))
		return 1
	}
	log := slogx.New(cfg.Matrix.Debug)
	perf, err := assistant.NewPerfLogger(cfg.Assistant.Perf.Log)
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
