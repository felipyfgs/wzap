package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	ServerHost  string
	AdminToken  string
	LogLevel    string
	Environment string

	DatabaseURL string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool

	NATSURL  string
	RedisURL string

	WALogLevel       string
	GlobalWebhookURL string
	ServerURL        string

	WSAuthMode string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:        getEnv("PORT", "8080"),
		ServerHost:  getEnv("SERVER_HOST", "0.0.0.0"),
		AdminToken:  getEnv("ADMIN_TOKEN", ""),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Environment: getEnv("ENVIRONMENT", "development"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9010"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", ""),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", ""),
		MinioBucket:    getEnv("MINIO_BUCKET", "wzap-media"),
		MinioUseSSL:    getEnvAsBool("MINIO_USE_SSL", false),

		NATSURL:  getEnv("NATS_URL", "nats://localhost:4222"),
		RedisURL: getEnv("REDIS_URL", ""),

		WALogLevel:       getEnv("WA_LOG_LEVEL", "INFO"),
		GlobalWebhookURL: getEnv("GLOBAL_WEBHOOK_URL", ""),
		ServerURL:        getEnv("SERVER_URL", "http://localhost:8080"),

		WSAuthMode: getEnv("WS_AUTH_MODE", "token"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return fallback
}
