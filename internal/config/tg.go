package config

type Telegram struct {
	BotToken string `yaml:"bot_token" env:"TG_BOT_TOKEN" env-required:"true"`
	ChatID   string `yaml:"chat_id"  env:"TG_CHAT_ID"  env-required:"true"`
}
