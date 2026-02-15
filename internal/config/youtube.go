package config

type Youtube struct {
	YoutubeToken string `env:"YOUTUBE_TOKEN,required"`
}
