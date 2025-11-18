package cfg

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort      string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	GrpsPort      string
	RedisAddr     string
	RedisPassword string
	KafkaBrokers  string
	KafkaTopic    string
	JWTSecret     string
}

func LoadConfig() Config {
	// Load .env if present (silently continue on error)
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using environment variables")
	}

	cfg := Config{
		HTTPPort:      os.Getenv("HTTP_PORT"),
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        os.Getenv("DB_PORT"),
		DBUser:        os.Getenv("DB_USER"),
		DBPassword:    os.Getenv("DB_PASSWORD"),
		DBName:        os.Getenv("DB_NAME"),
		GrpsPort:      os.Getenv("GRPC_PORT"),
		RedisAddr:     os.Getenv("REDIS_ADDR"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		KafkaBrokers:  os.Getenv("KAFKA_BROKERS"),
		KafkaTopic:    os.Getenv("KAFKA_TOPIC"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
	}

	return cfg
}
