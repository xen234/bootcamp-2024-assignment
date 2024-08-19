package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `yaml:"env" env:"ENV" env-default:"dev"`
	StoragePath string `yaml:"storage_path" env-required:"true"`
	HttpServer  struct {
		Address     string        `yaml:"address" env-default:"localhost:8081"`
		Timeout     time.Duration `yaml:"timeout" env-default:"5s"`
		IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
	}
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("CONFIG_PATH does not exist: %s", configPath)
	}

	var conf Config
	if err := cleanenv.ReadConfig(configPath, &conf); err != nil {
		log.Fatalf("Failed to read config: %s", err)
	}

	return &conf
}
