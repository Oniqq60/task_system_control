package cfg

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config содержит конфигурацию Notification Service
type Config struct {
	HTTPPort     string
	KafkaBrokers string
	KafkaTopic   string
	KafkaGroupID string
	LogLevel     string
	AuthGRPCAddr string
}

// LoadConfig загружает конфигурацию из окружения/.env
func LoadConfig() Config {
	_ = godotenv.Load(".env")

	cfg := Config{
		HTTPPort:     getEnvOrDefault("HTTP_PORT", "8083"),
		KafkaBrokers: os.Getenv("KAFKA_BROKERS"),
		KafkaTopic:   getEnvOrDefault("KAFKA_TOPIC", "task-events"),
		KafkaGroupID: getEnvOrDefault("KAFKA_GROUP_ID", "notification-service"),
		LogLevel:     getEnvOrDefault("LOG_LEVEL", "info"),
		AuthGRPCAddr: getEnvOrDefault("AUTH_GRPC_ADDR", "auth:9090"),
	}

	if cfg.KafkaBrokers == "" {
		log.Fatal("KAFKA_BROKERS is required")
	}

	return cfg
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
