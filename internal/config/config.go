package config

import (
	"os"

	"go.uber.org/zap"
)

type Config struct {
	Port string
	DB   DB
	Auth AuthConfig
}

type DB struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type AuthConfig struct {
	AdminToken string
	UserToken  string
}

func Load(log *zap.Logger) *Config {
	return &Config{
		Port: getEnv("APP_PORT", "8080", log),
		DB: DB{
			Host:     getEnv("DB_HOST", "localhost", log),
			Port:     getEnv("DB_PORT", "5432", log),
			User:     getEnv("DB_USER", "reviewer", log),
			Password: getEnv("DB_PASSWORD", "12341", log),
			Name:     getEnv("DB_NAME", "reviewer-pr-db", log),
			SSLMode:  getEnv("DB_SSLMODE", "disable", log),
		},
		Auth: AuthConfig{
			AdminToken: getEnv("ADMIN_TOKEN", "admin-token", log),
			UserToken:  getEnv("USER_TOKEN", "user-token", log),
		},
	}
}

func getEnv(key, defaultVal string, log *zap.Logger) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	if defaultVal != "" {
		log.Warn("Переменная окружения не установлена, используем значение по умолчанию",
			zap.String("key", key),
			zap.String("default", defaultVal),
		)
		return defaultVal
	}

	log.Error("Обязательная переменная окружения не установлена и значение по умолчанию не задано",
		zap.String("key", key),
	)
	panic("missing required environment variable: " + key)
}
