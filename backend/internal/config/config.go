package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all settings read from environment variables.
type Config struct {
	Env         string // "development" or "production"
	Port        string
	DatabaseURL string
	CORSOrigins []string // frontend origins allowed to call the API

	JWTSecret     string // key used to sign access tokens
	SecureCookies bool   // mark cookies "Secure" (HTTPS only); on in production
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

	rawOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if rawOrigins == "" {
		if cfg.Env == "production" {
			return nil, errors.New("CORS_ALLOWED_ORIGINS is required in production")
		}
		rawOrigins = "http://localhost:5173"
	}

	origins, err := parseOrigins(rawOrigins)
	if err != nil {
		return nil, err
	}
	if len(origins) == 0 {
		return nil, errors.New("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}
	cfg.CORSOrigins = origins

	cfg.JWTSecret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(cfg.JWTSecret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 characters; generate a random one (see README)")
	}
	if strings.HasPrefix(strings.ToLower(cfg.JWTSecret), "change-me") {
		return nil, errors.New("JWT_SECRET is still the placeholder value; generate a random one")
	}

	// Browsers only send "Secure" cookies over HTTPS, so this stays off for
	// local development on http://localhost.
	cfg.SecureCookies = cfg.Env == "production"

	return cfg, nil
}

// parseOrigins splits a comma-separated list and checks that every entry is a
// plain origin like http://localhost:5173 (scheme + host + optional port).
func parseOrigins(raw string) ([]string, error) {
	var origins []string

	for _, part := range strings.Split(raw, ",") {
		o := strings.TrimSpace(part)
		if o == "" {
			continue
		}
		if o == "*" {
			return nil, errors.New("CORS_ALLOWED_ORIGINS must list explicit origins, not \"*\"")
		}

		u, err := url.Parse(o)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") ||
			u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf(
				"CORS_ALLOWED_ORIGINS: %q is not a valid origin (expected something like http://localhost:5173, with no trailing slash or path)", o)
		}

		origins = append(origins, o)
	}

	return origins, nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
