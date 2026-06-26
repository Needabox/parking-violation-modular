package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort     string
	DatabaseURL    string
	JWTSecret      string
	JWTExpiration  time.Duration
	AllowedOrigins []string
}

func Load() (Config, error) {
	expHours := getEnvInt("JWT_EXPIRATION_HOURS", 24)
	cfg := Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/parking_violation?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		JWTExpiration:  time.Duration(expHours) * time.Hour,
		AllowedOrigins: splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:3000")),
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i <= len(value); i++ {
		if i == len(value) || value[i] == ',' {
			part := value[start:i]
			if part != "" {
				parts = append(parts, part)
			}
			start = i + 1
		}
	}
	return parts
}
