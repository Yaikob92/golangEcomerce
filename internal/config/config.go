package config

import (
	"fmt"
	"log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Env  string `env:"ENV" env-default:"development"`
	Port string `env:"PORT" env-default:"8080"`
	DB   DBConfig
	JWT  JWTConfig
}

type DBConfig struct {
	URL string `env:"DATABASE_URL" env-required:"true"`
}

type JWTConfig struct {
	Secret      string `env:"JWT_SECRET" env-required:"true"`
	ExpiryHours int    `env:"JWT_EXPIRY_HOURS" env-default:"24"`
}

// Load loads the configuration from .env and environment variables.
func Load() (*Config, error) {
	// Attempt to load from .env file, but don't fail if it's missing (e.g. in containerized production)
	if err := godotenv.Load(); err != nil {
		log.Println("Note: No .env file found or unable to load, falling back to system environment variables")
	}

	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read environment config: %w", err)
	}

	return &cfg, nil
}
