package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"tg_market/internal/application"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(log)

	if err := application.Run(ctx, log, cancel); err != nil {
		log.Error("application failed", "error", err)
		os.Exit(1)
	}

	log.Info("application stopped")
}
