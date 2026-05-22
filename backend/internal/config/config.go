package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr      string
	AdminUser     string
	AdminPassword string
	DBDriver      string
	DBDSN         string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisPrefix   string
}

func Load() Config {
	return Config{
		HTTPAddr:      envOrDefault("HTTP_ADDR", ":18100"),
		AdminUser:     envOrDefault("ADMIN_USER", "admin"),
		AdminPassword: envOrDefault("ADMIN_PASSWORD", "change-me-now"),
		DBDriver:      envOrDefault("DB_DRIVER", "mysql"),
		DBDSN:         envOrDefault("DB_DSN", "root:root@tcp(127.0.0.1:3306)/campaign_lottery_platform?parseTime=true&charset=utf8mb4&loc=Local"),
		RedisAddr:     envOrDefault("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       envIntOrDefault("REDIS_DB", 10),
		RedisPrefix:   envOrDefault("REDIS_PREFIX", "campaign:lottery:"),
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func envIntOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
