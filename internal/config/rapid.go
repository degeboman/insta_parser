package config

type Rapid struct {
	ApiKey string `env:"RAPID_APIKEY,required"`
}
