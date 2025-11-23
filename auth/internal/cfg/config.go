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
	JWTSecret     string
	JWTTTL        string
}

func LoadConfig() Config {

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
		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTTTL:        os.Getenv("JWT_TTL"),
	}

	return cfg
}
