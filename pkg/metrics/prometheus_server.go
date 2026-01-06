package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go-backend-example/pkg/contextx"
	"go-backend-example/pkg/logx"
)

const httpServerReadHeaderTimeout = 5 * time.Second

var logger = contextx.LoggerFromContextOrDefault //nolint:gochecknoglobals

type PrometheusServer struct {
	listenAddress string
}

func NewPrometheusServer(
	listenAddress string,
) PrometheusServer {
	return PrometheusServer{
		listenAddress: listenAddress,
	}
}

func (p PrometheusServer) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	httpServer := &http.Server{
		//nolint:exhaustruct
		Addr:              p.listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: httpServerReadHeaderTimeout,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		<-ctx.Done()

		if err := httpServer.Shutdown(context.WithoutCancel(ctx)); err != nil {
			logger(ctx).Error("httpServer.Shutdown", logx.Error(err))
		}
	}()

	logger(ctx).Info("prometheus server started", slog.String("address", p.listenAddress))

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("httpServer.ListenAndServe: %w", err)
	}

	logger(ctx).Info("prometheus server stopped")

	return nil
}
