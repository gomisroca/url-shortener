package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port string

	APIKey string

	BaseURL string

	CodeLength          int
	MaxCollisionRetries int

	// DefaultTTL of 0 means no expiry.
	DefaultTTL      time.Duration
	MaxClicksPerURL int

	CleanupInterval time.Duration
}

func Load() Config {
	return Config{
		Port:                getEnv("PORT", "8080"),
		APIKey:              getEnv("API_KEY", ""),
		BaseURL:             getEnv("BASE_URL", "http://localhost:8080"),
		CodeLength:          getEnvInt("CODE_LENGTH", 6),
		MaxCollisionRetries: getEnvInt("MAX_COLLISION_RETRIES", 5),
		DefaultTTL:          time.Duration(getEnvFloat("DEFAULT_TTL_SECONDS", 0)) * time.Second,
		MaxClicksPerURL:     getEnvInt("MAX_CLICKS_PER_URL", 1000),
		CleanupInterval:     time.Duration(getEnvFloat("CLEANUP_INTERVAL_SECONDS", 300)) * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}