package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	Telegram Telegram
	Postgres Postgres
	Bot      Bot
}

type Bot struct {
	Token   string `env:"BOT_TOKEN,required"`
	AdminID int64  `env:"BOT_ADMIN_ID,required"`
}

func Load() (Config, error) {
	_ = godotenv.Load()

	var config Config

	if err := env.Parse(&config); err != nil {
		return Config{}, fmt.Errorf("env.Parse: %w", err)
	}

	return config, nil
}

func correctNewlines(s string) string {
	return strings.NewReplacer(`"`, "", `\n`, "\n").Replace(s)
}
