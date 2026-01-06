package application

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	"git.appkode.ru/pub/go/live/clock"
	"git.appkode.ru/pub/go/live/xidgenerator"
	"git.appkode.ru/pub/go/metrics"
	"git.appkode.ru/pub/go/metrics/field"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"go-backend-example/internal/config"
	"go-backend-example/internal/domain/service/example"
	"go-backend-example/internal/infrastructure/persistence"
	"go-backend-example/internal/server"
	"go-backend-example/pkg/application/connectors"
	"go-backend-example/pkg/application/modules"
	"go-backend-example/pkg/contextx"
	"go-backend-example/pkg/logx"
	"go-backend-example/pkg/middlewarex"
)

var logger = contextx.LoggerFromContextOrDefault //nolint:gochecknoglobals

type App struct {
	cfg            config.Config
	clock          clock.Clock
	xidGenerator   xidgenerator.Generator
	metrics        *metrics.Metrics
	slog           *connectors.Slog
	postgres       *connectors.Postgres
	probeServer    modules.ProbeServer
	metricServer   modules.MetricServer
	httpServer     modules.HTTPServer
	exampleService example.Service
	exampleRepo    persistence.Example
}

func New(appVersion string) App { //nolint:funlen
	const appName = "go-backend-example"

	cfg := lo.Must(config.Load())

	return App{
		cfg: cfg,
		slog: &connectors.Slog{
			Name:    appName,
			Version: appVersion,
			Debug:   cfg.Debug,
		},
		clock:        clock.New(),
		xidGenerator: xidgenerator.New(),
		metrics: metrics.NewMetrics(
			field.NewName(appName),
			field.NewEmptyName(),
			field.NewVersion(appVersion),
		),
		postgres: &connectors.Postgres{
			DSN:             cfg.Postgres.DSN,
			MaxIdleConns:    cfg.Postgres.MaxIdleConns,
			MaxOpenConns:    cfg.Postgres.MaxOpenConns,
			ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
		},
		metricServer: modules.MetricServer{
			ListenAddress: cfg.Prometheus.ListenAddress,
		},
		probeServer: modules.ProbeServer{
			Name:          appName,
			Version:       appVersion,
			ListenAddress: cfg.Probe.ListenAddress,
		},
		httpServer: modules.HTTPServer{
			ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
		},
	}
}

func (app App) shutdown(ctx context.Context) {
	app.postgres.Close(ctx)
}

func (app App) Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	defer stop()

	ctx = contextx.WithLogger(ctx, app.slog.Logger(ctx))

	defer app.shutdown(ctx)

	logger(ctx).Info("config", slog.Any("config", app.cfg))

	app.exampleRepo = persistence.NewExample(app.postgres.Client(ctx))
	app.exampleService = example.NewService(app.exampleRepo)

	g, ctx := errgroup.WithContext(ctx)

	app.httpServer.Run(ctx, g, app.newHTTPServer(ctx, app.exampleService))
	app.metricServer.Run(ctx, g)
	app.probeServer.Run(ctx, g)

	if err := g.Wait(); err != nil {
		return fmt.Errorf("g.Wait: %w", err)
	}

	return nil
}

func (app App) newHTTPServer(ctx context.Context, exampleService example.Service) *http.Server { //nolint:funlen,maintidx
	router := chi.NewRouter()

	router.Use(
		middleware.RealIP,
		middlewarex.TraceID,
		middlewarex.Logger,
		middlewarex.RequestLogging(app.newSensitiveDataMasker(), app.cfg.Log.FieldMaxLen),
		middlewarex.ResponseLogging(app.newSensitiveDataMasker(), app.cfg.Log.FieldMaxLen),
		middlewarex.Recovery,
	)

	server.NewServer(
		server.NewExampleServer(exampleService),
	).RegisterRoutes(router)

	return &http.Server{
		//nolint:exhaustruct
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		Addr:              app.cfg.HTTP.ListenAddress,
		WriteTimeout:      app.cfg.HTTP.WriteTimeout,
		ReadTimeout:       app.cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: app.cfg.HTTP.ReadTimeout,
		IdleTimeout:       app.cfg.HTTP.IdleTimeout,
		Handler:           router,
	}
}

func (app App) newSensitiveDataMasker() logx.SensitiveDataMaskerInterface { //nolint:ireturn
	if !app.cfg.Log.SensitiveDataMasker.Enabled {
		return logx.NewNopSensitiveDataMasker()
	}

	return logx.NewSensitiveDataMasker()
}
