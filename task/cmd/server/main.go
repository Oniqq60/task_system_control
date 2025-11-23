package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	pb "github.com/Oniqq60/task_system_control/gen/proto/task"
	"github.com/Oniqq60/task_system_control/task/internal/cfg"
	"github.com/Oniqq60/task_system_control/task/internal/task"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	conf := cfg.LoadConfig()
	logger := log.New(os.Stdout, "[task] ", log.LstdFlags|log.Lmicroseconds)

	db := mustConnectDB(conf)
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatalf("failed to access sql DB: %v", err)
	}
	defer sqlDB.Close()

	brokers := splitCSV(conf.KafkaBrokers)
	if len(brokers) == 0 {
		logger.Fatal("KAFKA_BROKERS must be set")
	}
	if conf.KafkaTopic == "" {
		logger.Fatal("KAFKA_TOPIC must be set")
	}
	producer := task.NewKafkaProducer(brokers, conf.KafkaTopic)
	defer producer.Close()

	repo := task.NewRepository(db)
	service := task.NewTaskService(repo, producer)
	grpcHandler := task.NewGrpcHandler(service)

	httpMux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:    ":" + pickPort(conf.HTTPPort, "8081"),
		Handler: applyHTTPMiddleware(httpMux),
	}

	grpcListener, err := net.Listen("tcp", ":"+pickPort(conf.GrpsPort, "9091"))
	if err != nil {
		logger.Fatalf("failed to listen on gRPC port: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterTaskServiceServer(grpcServer, grpcHandler)

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
	logger.Println("task service stopped")
}

func mustConnectDB(conf cfg.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		conf.DBHost,
		conf.DBPort,
		conf.DBUser,
		conf.DBPassword,
		conf.DBName,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to init sql DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func applyHTTPMiddleware(mux *http.ServeMux) http.Handler {
	handler := http.Handler(mux)
	handler = task.RequestSizeLimitMiddleware(5 << 20)(handler)
	handler = task.CORSMiddleware(handler)
	handler = task.SecurityHeadersMiddleware(handler)
	return handler
}

func pickPort(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
