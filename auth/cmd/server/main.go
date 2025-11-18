package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	iauth "github.com/Oniqq60/task_system_control/auth/internal/auth"
	"github.com/Oniqq60/task_system_control/auth/internal/cfg"
	pb "github.com/Oniqq60/task_system_control/gen/proto/auth"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := cfg.LoadConfig()

	// Build Postgres DSN. If you manage SSL externally, set sslmode accordingly via env in future.
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

	// Connect Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,     // e.g. "localhost:6379"
		Password: cfg.RedisPassword, // empty if no auth
		DB:       0,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping error: %v", err)
	}
	log.Println("redis connected")

	// Init GORM for repository layer
	gormDB, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn}), &gorm.Config{})
	if err != nil {
		log.Fatalf("gorm open error: %v", err)
	}

	// JWT config
	jwtSecret := []byte(cfg.JWTSecret)
	if len(jwtSecret) < 32 {
		log.Fatalf("JWT_SECRET must be at least 32 characters long for security")
	}
	ttlSeconds := int64(3600)
	if cfg.JWTTTL != "" {
		if v, err := strconv.Atoi(cfg.JWTTTL); err == nil {
			ttlSeconds = int64(v)
		}
	}

	// Wire
	repo := iauth.NewRepository(gormDB)
	service := iauth.NewUserService(repo, jwtSecret, ttlSeconds)
	handler := iauth.NewUserHandler(service, jwtSecret, rdb)
	grpcHandler := iauth.NewGrpcHandler(service, jwtSecret, rdb)

	// HTTP Server with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", handler.Register)
	mux.HandleFunc("/auth/login", handler.Login)
	mux.HandleFunc("/auth/logout", handler.Logout)
	mux.HandleFunc("/auth/profile", handler.Profile)

	// Применяем middleware
	handlerWithMiddleware := iauth.SecurityHeadersMiddleware(
		iauth.CORSMiddleware(
			iauth.RequestSizeLimitMiddleware(1024 * 1024)(mux), // 1MB limit
		),
	)

	httpPort := cfg.HTTPPort
	if httpPort == "" {
		httpPort = "8080"
	}

	// gRPC Server
	grpcPort := cfg.GrpsPort
	if grpcPort == "" {
		grpcPort = "9090"
	}

	grpcListener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen on gRPC port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcServer, grpcHandler)

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
