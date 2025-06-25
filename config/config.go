package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type JWTConfig struct {
	Secret   string
	Issuer   string
	Audience string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type Config struct {
	Env         string
	Port        string
	DatabaseURL string
	FrontendURL string
	JWT         JWTConfig
	SMTP        SMTPConfig
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	cfg := &Config{}

	cfg.Env = getEnvOrDefault("ENV", "development")
	cfg.Port = getEnvOrDefault("PORT", "8080")

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	cfg.FrontendURL = getEnvOrDefault("FRONTEND_URL", getEnvOrDefault("APP_BASE_URL", "http://localhost:3000"))

	cfg.JWT.Secret = os.Getenv("JWT_SECRET")
	if cfg.JWT.Secret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	cfg.JWT.Issuer = getEnvOrDefault("JWT_ISSUER", "yanemarket")
	cfg.JWT.Audience = getEnvOrDefault("JWT_AUDIENCE", "yanemarket-api")

	cfg.SMTP.Host = getEnvOrDefault("SMTP_HOST", "smtp.gmail.com")
	portStr := getEnvOrDefault("SMTP_PORT", "587")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 587
	}
	cfg.SMTP.Port = port
	cfg.SMTP.Username = os.Getenv("SMTP_USERNAME")
	cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")
	cfg.SMTP.From = getEnvOrDefault("SMTP_FROM", "noreply@yanemarket.com")

	return cfg
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
