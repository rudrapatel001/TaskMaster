package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all settings read from environment variables.
type Config struct {
	Env         string // "development" or "production"
	Port        string
	DatabaseURL string
}

// Load reads configuration from the environment, using backend/.env as a
// convenience in development. Variables that are already set in the real
// environment always win over the .env file.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading .env file: %w", err)
	}

	cfg := &Config{
		Env:         getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.Env != "development" && cfg.Env != "production" {
		return nil, fmt.Errorf("APP_ENV must be \"development\" or \"production\", got %q", cfg.Env)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
