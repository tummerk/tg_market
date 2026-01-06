package config

import "time"

type Postgres struct {
	DSN             string        `env:"PG_DSN,notEmpty" json:"-"`
	MaxIdleConns    int           `env:"PG_MAX_IDLE_CONNS" envDefault:"5"`
	MaxOpenConns    int           `env:"PG_MAX_OPEN_CONNS" envDefault:"5"`
	ConnMaxLifetime time.Duration `env:"PG_CONN_MAX_LIFETIME" envDefault:"5m"`
}
