package cfg

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort         string
	GRPCPort         string
	MongoURI         string
	MongoDatabase    string
	MongoCollection  string
	MinioEndpoint    string
	MinioAccessKey   string
	MinioSecretKey   string
	MinioUseSSL      bool
	MinioBucket      string
	MaxFileSizeBytes int64
	JWTSecret        string
	RedisAddr        string
	RedisPassword    string
}

func LoadConfig() Config {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using environment variables")
	}

	cfg := Config{
		HTTPPort:        os.Getenv("HTTP_PORT"),
		GRPCPort:        os.Getenv("GRPC_PORT"),
		MongoURI:        os.Getenv("MONGODB_URI"),
		MongoDatabase:   os.Getenv("MONGODB_DATABASE"),
		MongoCollection: os.Getenv("MONGODB_COLLECTION"),
		MinioEndpoint:   os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:  os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:  os.Getenv("MINIO_SECRET_KEY"),
		MinioBucket:     os.Getenv("MINIO_BUCKET"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		RedisAddr:       os.Getenv("REDIS_ADDR"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
	}

	// MINIO_USE_SSL optional
	if os.Getenv("MINIO_USE_SSL") == "true" || os.Getenv("MINIO_USE_SSL") == "1" {
		cfg.MinioUseSSL = true
	}

	// MAX_FILE_SIZE optional, default 10MB
	if maxStr := os.Getenv("MAX_FILE_SIZE"); maxStr != "" {
		if v, err := strconv.ParseInt(maxStr, 10, 64); err == nil {
			cfg.MaxFileSizeBytes = v
		}
	}
	if cfg.MaxFileSizeBytes == 0 {
		cfg.MaxFileSizeBytes = 10 * 1024 * 1024 // 10 MB
	}

	return cfg
}
