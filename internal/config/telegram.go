package config

type Telegram struct {
	ApiID    int    `env:"TG_API_ID,required"`
	ApiHash  string `env:"TG_API_HASH,required"`
	Phone    string `env:"TG_PHONE,required"`
	Password string `env:"TG_PASSWORD"`
	BotToken string `env:"TG_BOT_TOKEN"`
}
