package modules

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"golang.org/x/sync/errgroup"
)

type AsynqQueues map[string]int

type AsynqHandler struct {
	Pattern string
	Handle  func(context.Context, *asynq.Task) error
}

type AsynqServer struct {
	RedisUsername string
	RedisPassword string
	RedisAddress  string
	RedisDB       int
}

func (s AsynqServer) Run(
	ctx context.Context,
	g *errgroup.Group,
	queues AsynqQueues,
	handlers ...AsynqHandler,
) {
	g.Go(func() error {
		redisConnection := asynq.RedisClientOpt{
			Addr:     s.RedisAddress,
			Username: s.RedisUsername,
			Password: s.RedisPassword,
			DB:       s.RedisDB,
		}

		worker := asynq.NewServer(redisConnection, asynq.Config{
			BaseContext: func() context.Context { return ctx },
			Queues:      queues,
		})

		mux := asynq.NewServeMux()

		for _, h := range handlers {
			mux.HandleFunc(h.Pattern, h.Handle)
		}

		logger(ctx).Info("asynq server started", slog.String("redis-address", s.RedisAddress), slog.Int("redis-db", s.RedisDB))

		if err := worker.Run(mux); err != nil {
			return fmt.Errorf("asynqServer.Run: %w", err)
		}

		logger(ctx).Info("asynq server stopped", slog.String("redis-address", s.RedisAddress), slog.Int("redis-db", s.RedisDB))

		return nil
	})
}
