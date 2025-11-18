package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	pb "github.com/Oniqq60/task_system_control/gen/proto/task"
	"github.com/Oniqq60/task_system_control/task/internal/cfg"
	itask "github.com/Oniqq60/task_system_control/task/internal/task"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := cfg.LoadConfig()

	if len(cfg.JWTSecret) < 32 {
		log.Fatalf("JWT_SECRET must be at least 32 characters long for security")
	}

	// Build Postgres DSN
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	// Connect Postgres using pgx stdlib driver
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("postgres open error: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("postgres ping error: %v", err)
	}
	log.Println("postgres connected")

	var redisClient *redis.Client
	if cfg.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       0,
		})
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			log.Fatalf("redis ping error: %v", err)
		}
		log.Println("redis connected")
		defer redisClient.Close()
	}

	// Init GORM for repository layer
	gormDB, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn}), &gorm.Config{})
	if err != nil {
		log.Fatalf("gorm open error: %v", err)
	}

	// Подключение Kafka producer
	var kafkaProducer itask.KafkaProducer
	if cfg.KafkaBrokers != "" && cfg.KafkaTopic != "" {
		brokers := strings.Split(cfg.KafkaBrokers, ",")
		for i := range brokers {
			brokers[i] = strings.TrimSpace(brokers[i])
		}
		kafkaProducer = itask.NewKafkaProducer(brokers, cfg.KafkaTopic)
		log.Println("Kafka producer connected")
		defer kafkaProducer.Close()
	} else {
		log.Println("Kafka not configured, events will not be sent")
	}

	// Wire
	repo := itask.NewRepository(gormDB)
	service := itask.NewTaskService(repo, kafkaProducer)
	handler := itask.NewHandler(service, []byte(cfg.JWTSecret), redisClient)
	grpcHandler := itask.NewGrpcHandler(service)

	// HTTP Server
	mux := http.NewServeMux()
	mux.HandleFunc("POST /task", handler.CreateTask)
	mux.HandleFunc("PATCH /task/{id}", handler.UpdateTask)
	mux.HandleFunc("GET /task", handler.TaskList)

	// Применяем middleware
	handlerWithMiddleware := itask.SecurityHeadersMiddleware(
		itask.CORSMiddleware(
			itask.RequestSizeLimitMiddleware(1024 * 1024)(mux), // 1MB limit
		),
	)

	httpPort := cfg.HTTPPort
	if httpPort == "" {
		httpPort = "8081"
	}

	// gRPC Server
	grpcPort := cfg.GrpsPort
	if grpcPort == "" {
		grpcPort = "9091"
	}

	grpcListener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on gRPC port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTaskServiceServer(grpcServer, grpcHandler)

	// Запускаем gRPC сервер в отдельной goroutine
	go func() {
		log.Printf("gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Запускаем HTTP сервер
	srv := &http.Server{Addr: ":" + httpPort, Handler: handlerWithMiddleware}
	log.Printf("http server listening on :%s", httpPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server error: %v", err)
	}
}
