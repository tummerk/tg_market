package contextx

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

type contextKeyLogger struct{}

var DefaultLogger = slog.New(slog.NewTextHandler(os.Stdout, nil)) //nolint:gochecknoglobals

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKeyLogger{}, logger)
}

func LoggerFromContext(ctx context.Context) (*slog.Logger, error) {
	logger, ok := ctx.Value(contextKeyLogger{}).(*slog.Logger)
	if !ok {
		return nil, fmt.Errorf("logger: %w", ErrNoValue)
	}

	return logger, nil
}

func LoggerFromContextOrDefault(ctx context.Context) *slog.Logger {
	logger, err := LoggerFromContext(ctx)
	if err != nil {
		return DefaultLogger
	}

	return logger
}
