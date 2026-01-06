package connectors

import (
	"context"
	"log/slog"
	"net/url"
	"sync"
	"tg_market/pkg/logx"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // golang postgres driver
	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"
)

type Postgres struct {
	value           *sqlx.DB
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	init            sync.Once
}

func (p *Postgres) Client(ctx context.Context) *sqlx.DB {
	p.init.Do(func() {
		p.value = lo.Must(sqlx.ConnectContext(ctx, "pgx", p.DSN))

		p.value.SetMaxOpenConns(p.MaxOpenConns)
		p.value.SetMaxIdleConns(p.MaxIdleConns)
		p.value.SetConnMaxLifetime(p.ConnMaxLifetime)

		logger(ctx).Info(
			"postgres connected",
			slog.String("database", lo.Must(url.Parse(p.DSN)).Path),
		)
	})

	return p.value
}

func (p *Postgres) Close(ctx context.Context) {
	if err := p.value.Close(); err != nil {
		logger(ctx).Error("postgresClient.Close", logx.Error(err))
	}

	logger(ctx).Info(
		"postgres disconnected",
		slog.String("database", lo.Must(url.Parse(p.DSN)).Path),
	)
}
