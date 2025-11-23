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

	appcfg "github.com/Oniqq60/task_system_control/document/internal/cfg"
	"github.com/Oniqq60/task_system_control/document/internal/document"
	pb "github.com/Oniqq60/task_system_control/gen/proto/document"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

func main() {
	conf := appcfg.LoadConfig()
	logger := log.New(os.Stdout, "[document] ", log.LstdFlags|log.Lmicroseconds)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mongoClient := mustConnectMongo(ctx, conf.MongoURI)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(shutdownCtx); err != nil {
			logger.Printf("mongo disconnect error: %v", err)
		}
	}()

	collection := mongoClient.Database(conf.MongoDatabase).Collection(conf.MongoCollection)
	repo := document.NewRepository(collection)

	storage, err := document.NewMinioStorage(
		conf.MinioEndpoint,
		conf.MinioAccessKey,
		conf.MinioSecretKey,
		conf.MinioUseSSL,
		conf.MinioBucket,
	)
	if err != nil {
		logger.Fatalf("failed to connect to MinIO: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Password: conf.RedisPassword,
	})
	defer redisClient.Close()

	service := document.NewService(repo, storage)
	authorizer := document.NewAuthorizer([]byte(conf.JWTSecret), redisClient)
	grpcHandler := document.NewGrpcHandler(service, conf.MaxFileSizeBytes, authorizer)

	httpMux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:    ":" + pickPort(conf.HTTPPort, "8082"),
		Handler: applyHTTPMiddleware(httpMux),
	}

	grpcListener, err := net.Listen("tcp", ":"+pickPort(conf.GRPCPort, "9093"))
	if err != nil {
		logger.Fatalf("failed to listen on gRPC port: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterDocumentServiceServer(grpcServer, grpcHandler)

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
	logger.Println("document service stopped")
}

func mustConnectMongo(ctx context.Context, uri string) *mongo.Client {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}
	return client
}

func applyHTTPMiddleware(handler http.Handler) http.Handler {
	h := document.CORSMiddleware(handler)
	h = document.SecurityHeadersMiddleware(h)
	return h
}

func pickPort(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
