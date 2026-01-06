package connectors

import (
	"context"
	"log/slog"
	"sync"
	"tg_market/pkg/logx"

	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
)

type Redis struct {
	value              *redis.Client
	Username           string
	Password           string
	Address            string
	DatabaseNumber     int
	PoolSize           int
	MinIdleConnections int
	MaxIdleConnections int
	init               sync.Once
}

func (r *Redis) Client(ctx context.Context) *redis.Client {
	r.init.Do(func() {
		r.value = redis.NewClient(&redis.Options{
			//nolint:exhaustruct
			Network:      "tcp",
			Addr:         r.Address,
			Username:     r.Username,
			Password:     r.Password,
			DB:           r.DatabaseNumber,
			PoolSize:     r.PoolSize,
			MinIdleConns: r.MinIdleConnections,
			MaxIdleConns: r.MaxIdleConnections,
		})

		lo.Must0(r.value.Ping(ctx).Err())

		logger(ctx).Info(
			"redis connected",
			slog.String("address", r.Address),
			slog.Int("database", r.DatabaseNumber),
		)
	})

	return r.value
}

func (r *Redis) Close(ctx context.Context) {
	if err := r.value.Close(); err != nil {
		logger(ctx).Error("redisClient.Close", logx.Error(err))
	}

	logger(ctx).Info(
		"redis disconnected",
		slog.String("address", r.Address),
		slog.Int("database", r.DatabaseNumber),
	)
}
