package config

import (
	"fmt"
	"os"
)

// Config holds runtime configuration sourced from the environment.
type Config struct {
	Port        string
	DatabaseURL string
}

// Load reads configuration from environment variables, applying sane
// defaults for local development.
func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/order_service?sslmode=disable"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Addr returns the address the HTTP server should bind to.
func (c Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}
