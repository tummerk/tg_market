package config

import "time"

type Telegram struct {
	ApiID           int    `env:"TG_API_ID,required"`
	ApiHash         string `env:"TG_API_HASH,required"`
	RatePerClientMs int    `env:"RATE_PER_CLIENT_MS,required"`
}

func (t *Telegram) GetRatePerClient() time.Duration {
	return time.Duration(t.RatePerClientMs) * time.Millisecond
}
