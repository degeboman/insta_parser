package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Rapid                  Rapid
	VK                     VK
	GoogleDriveCredentials GoogleDriveCredentials
	Youtube                Youtube
}

func MustLoad() Config {
	var cfg Config

	if err := godotenv.Load(".env"); err != nil {
		log.Printf("cannot load env file %v", err)
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("cannot read env: %s", err)
	}

	return cfg
}

func MustLoadForTest() Config {
	var cfg Config

	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("cannot load env file %v", err)
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("cannot read env: %s", err)
	}

	return cfg
}
