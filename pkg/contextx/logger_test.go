package contextx_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"go-backend-example/pkg/contextx"
)

func TestLogger(t *testing.T) {
	rq := require.New(t)
	ctx := context.Background()

	var testLoggerNil *slog.Logger

	testLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	logger, err := contextx.LoggerFromContext(ctx)
	rq.Equal(testLoggerNil, logger)
	rq.ErrorIs(err, contextx.ErrNoValue)
	rq.ErrorContains(err, "logger: no value in context")

	ctx = contextx.WithLogger(ctx, testLogger)

	logger, err = contextx.LoggerFromContext(ctx)
	rq.Equal(testLogger, logger)
	rq.NoError(err)

	logger.With(slog.String("key", "value"))
}
