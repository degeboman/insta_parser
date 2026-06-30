package config

type VK struct {
	Token           string `env:"VK_TOKEN,required"`
	TokenKateMobile string `env:"VK_TOKEN_KATE_MOBILE,required"`
}
