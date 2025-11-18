package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Oniqq60/task_system_control/document/internal/cfg"
	idoc "github.com/Oniqq60/task_system_control/document/internal/document"
	pb "github.com/Oniqq60/task_system_control/gen/proto/document"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

func main() {
	config := cfg.LoadConfig()
	if len(config.JWTSecret) < 32 {
		log.Fatalf("JWT_SECRET must be at least 32 characters long for security")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoURI))
	if err != nil {
		log.Fatalf("failed to connect mongo: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("mongo disconnect error: %v", err)
		}
	}()

	coll := mongoClient.Database(config.MongoDatabase).Collection(config.MongoCollection)

	storage, err := idoc.NewMinioStorage(
		config.MinioEndpoint,
		config.MinioAccessKey,
		config.MinioSecretKey,
		config.MinioUseSSL,
		config.MinioBucket,
	)
	if err != nil {
		log.Fatalf("failed to init minio: %v", err)
	}

	var redisClient *redis.Client
	if config.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       0,
		})
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			log.Fatalf("failed to connect redis: %v", err)
		}
	}

	repo := idoc.NewRepository(coll)
	service := idoc.NewService(repo, storage)

	handler := idoc.NewHandler(service, []byte(config.JWTSecret), config.MaxFileSizeBytes, redisClient)
	httpMux := handler.Routes()

	// Применяем middleware
	handlerWithMiddleware := idoc.SecurityHeadersMiddleware(
		idoc.CORSMiddleware(httpMux),
	)

	httpPort := config.HTTPPort
	if httpPort == "" {
		httpPort = "8082"
	}
	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: handlerWithMiddleware,
	}

	grpcPort := config.GRPCPort
	if grpcPort == "" {
		grpcPort = "9093"
	}

	grpcListener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen grpc: %v", err)
	}

	grpcServer := grpc.NewServer()
	authorizer := idoc.NewAuthorizer([]byte(config.JWTSecret), redisClient)
	grpcHandler := idoc.NewGrpcHandler(service, config.MaxFileSizeBytes, authorizer)
	pb.RegisterDocumentServiceServer(grpcServer, grpcHandler)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Printf("HTTP server listening on :%s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("gRPC server listening on :%s", grpcPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("grpc server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
	grpcServer.GracefulStop()

	wg.Wait()
}
