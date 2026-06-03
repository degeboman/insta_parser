package config

type ValKey struct {
	Host     string `env:"VALKEY_HOST,required"`
	Password string `env:"VALKEY_PASSWORD,required"`
	Username string `env:"VALKEY_USERNAME,required"`
}
