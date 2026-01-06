package modules

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"go-backend-example/pkg/logx"
)

// HTTPServer модуль, ответственный за запуск и остановку HTTP-сервера
// (graceful shutdown).
type HTTPServer struct {
	ShutdownTimeout time.Duration
}

func (h HTTPServer) Run(
	ctx context.Context,
	g *errgroup.Group,
	httpServer *http.Server,
) {
	g.Go(func() error {
		go func() {
			<-ctx.Done()

			ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), h.ShutdownTimeout) //nolint:govet
			defer cancel()

			if err := httpServer.Shutdown(ctx); err != nil {
				logger(ctx).Error("server.Shutdown", logx.Error(err))
			}
		}()

		logger(ctx).Info("http server started", slog.String("address", httpServer.Addr))

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("httpServer.ListenAndServe: %w", err)
		}

		logger(ctx).Info("http server stopped", slog.String("address", httpServer.Addr))

		return nil
	})
}
