// Package config loads app configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr         string // HTTP listen address
	DatabasePath string // sqlite file path
	UploadsDir   string // runtime-writable dir for uploaded media
	SessionKey   []byte // HMAC key for signing session/CSRF tokens
	Env          string // "dev" or "prod" — gates cookie Secure flag
}

func Load() (Config, error) {
	cfg := Config{
		Addr:         getEnv("ADDR", ":8080"),
		DatabasePath: getEnv("DATABASE_PATH", "boutique.db"),
		UploadsDir:   getEnv("UPLOADS_DIR", "uploads"),
		Env:          getEnv("ENV", "dev"),
	}

	key := os.Getenv("SESSION_KEY")
	if key == "" {
		if cfg.Env == "prod" {
			return Config{}, fmt.Errorf("SESSION_KEY must be set in prod")
		}
		key = "dev-insecure-session-key-change-me"
	}
	cfg.SessionKey = []byte(key)

	return cfg, nil
}

func (c Config) IsProd() bool {
	return c.Env == "prod"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
