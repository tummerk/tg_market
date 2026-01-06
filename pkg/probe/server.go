package probe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"

	"go-backend-example/pkg/contextx"
	"go-backend-example/pkg/logx"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary //nolint:gochecknoglobals // skip

const httpServerReadHeaderTimeout = 5 * time.Second

var logger = contextx.LoggerFromContextOrDefault //nolint:gochecknoglobals

type Server struct {
	listenAddress string
	state         []byte
}

type Options struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewServer(
	listenAddress string,
	options Options,
) Server {
	stateJSON, _ := json.Marshal(options) //nolint:errcheck,errchkjson

	return Server{
		listenAddress: listenAddress,
		state:         stateJSON,
	}
}

func (s Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", s.handlerHealthz)
	mux.HandleFunc("/ready", s.handlerReady)

	httpServer := &http.Server{
		//nolint:exhaustruct
		Addr:              s.listenAddress,
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

	logger(ctx).Info("probe server started", slog.String("address", s.listenAddress))

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("httpServer.ListenAndServe: %w", err)
	}

	logger(ctx).Info("probe server stopped")

	return nil
}

func (s Server) handlerHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(s.state) //nolint:errcheck
}

func (s Server) handlerReady(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(s.state) //nolint:errcheck
}
