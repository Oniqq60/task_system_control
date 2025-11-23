package cfg

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort             string
	AuthGRPCAddr         string
	TaskGRPCAddr         string
	DocumentGRPCAddr     string
	JWTSecret            string
	RateLimitRequests    int
	RateLimitWindow      time.Duration
	AllowedCORSOrigins   []string
	ShutdownGracePeriod  time.Duration
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	ForwardResponseLimit int64
}

func Load() (Config, error) {
	_ = godotenv.Load(".env")

	cfg := Config{
		HTTPPort:             getEnv("HTTP_PORT", "8084"),
		AuthGRPCAddr:         getEnv("AUTH_GRPC_ADDR", "auth:9090"),
		TaskGRPCAddr:         getEnv("TASK_GRPC_ADDR", "task:9091"),
		DocumentGRPCAddr:     getEnv("DOCUMENT_GRPC_ADDR", "document:9093"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		RateLimitRequests:    getEnvInt("RATE_LIMIT_REQUESTS", 60),
		RateLimitWindow:      getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		AllowedCORSOrigins:   parseCSVEnv("ALLOWED_ORIGINS"),
		ShutdownGracePeriod:  getEnvDuration("SHUTDOWN_GRACE_PERIOD", 10*time.Second),
		ReadTimeout:          getEnvDuration("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:         getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:          getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		ForwardResponseLimit: getEnvInt64("FORWARD_RESPONSE_LIMIT", 10*1024*1024),
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			return parsed
		}
	}
	return fallback
}

func parseCSVEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
