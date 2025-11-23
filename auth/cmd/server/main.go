package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Oniqq60/task_system_control/auth/internal/auth"
	appcfg "github.com/Oniqq60/task_system_control/auth/internal/cfg"
	pb "github.com/Oniqq60/task_system_control/gen/proto/auth/v1"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := appcfg.LoadConfig()
	logger := log.New(os.Stdout, "[auth] ", log.LstdFlags|log.Lmicroseconds)

	db := mustInitDB(cfg)
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatalf("failed to access sql DB: %v", err)
	}
	defer sqlDB.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
	defer redisClient.Close()

	jwtSecret := []byte(cfg.JWTSecret)
	jwtTTL := parseTTL(cfg.JWTTTL, 3600)

	repo := auth.NewRepository(db)
	userService := auth.NewUserService(repo, jwtSecret, jwtTTL)

	grpcHandler := auth.NewGrpcHandler(userService, jwtSecret, redisClient)

	httpMux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:    ":" + pickPort(cfg.HTTPPort, "8080"),
		Handler: applyHTTPMiddleware(httpMux),
	}

	grpcListener, err := net.Listen("tcp", ":"+pickPort(cfg.GrpsPort, "9090"))
	if err != nil {
		logger.Fatalf("failed to listen on gRPC port: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterAuthServiceServer(grpcServer, grpcHandler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)

	go func() {
		logger.Printf("HTTP server listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	go func() {
		logger.Printf("gRPC server listening on %s", grpcListener.Addr().String())
		if err := grpcServer.Serve(grpcListener); err != nil {
			errCh <- fmt.Errorf("grpc server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")
	case err := <-errCh:
		logger.Printf("server error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Printf("http shutdown error: %v", err)
	}
	grpcServer.GracefulStop()
	logger.Println("auth service stopped")
}

func mustInitDB(cfg appcfg.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to init sql DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db
}

func applyHTTPMiddleware(mux *http.ServeMux) http.Handler {
	handler := http.Handler(mux)
	handler = auth.RequestSizeLimitMiddleware(5 << 20)(handler)
	handler = auth.CORSMiddleware(handler)
	handler = auth.SecurityHeadersMiddleware(handler)
	return handler
}

func pickPort(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseTTL(value string, fallback int64) int64 {
	if secs, err := strconv.ParseInt(value, 10, 64); err == nil && secs > 0 {
		return secs
	}
	return fallback
}
