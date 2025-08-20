package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL  string
	JWTSecret    string
	Environment  string
	CORSOrigins  []string
	RateLimitRPS int
	TokenExpiry  time.Duration
	DatabasePool DatabasePoolConfig
}

type DatabasePoolConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnvOrDefault("DATABASE_URL", "postgres://postgres:password@localhost:5432/llm_inferra?sslmode=disable"),
		JWTSecret:   getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),
		CORSOrigins: []string{
			getEnvOrDefault("FRONTEND_URL", "http://localhost:5173"),
		},
		RateLimitRPS: getEnvIntOrDefault("RATE_LIMIT_RPS", 100),
		TokenExpiry:  time.Hour * 24 * 7, // 7 days
		DatabasePool: DatabasePoolConfig{
			MaxIdleConns:    getEnvIntOrDefault("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getEnvIntOrDefault("DB_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getDurationFromEnvOrDefault("DB_CONN_MAX_LIFETIME", time.Hour),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationFromEnvOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
