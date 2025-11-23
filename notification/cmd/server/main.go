package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Oniqq60/task_system_control/notification/internal/authclient"
	"github.com/Oniqq60/task_system_control/notification/internal/cfg"
	"github.com/Oniqq60/task_system_control/notification/internal/notification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conf := cfg.LoadConfig()
	logger := log.New(os.Stdout, "[notification] ", log.LstdFlags|log.Lmicroseconds)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	brokers := splitCSV(conf.KafkaBrokers)
	if len(brokers) == 0 {
		logger.Fatal("KAFKA_BROKERS must be set")
	}
	if conf.KafkaTopic == "" {
		logger.Fatal("KAFKA_TOPIC must be set")
	}

	conn, err := grpc.DialContext(
		ctx,
		conf.AuthGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		logger.Fatalf("failed to connect to auth gRPC: %v", err)
	}
	defer conn.Close()

	authClient := authclient.New(conn, 5*time.Second)
	notifier := notification.NewLogNotifier(logger)
	handler := notification.NewEventHandler(notifier, authClient)
	consumer := notification.NewKafkaConsumer(brokers, conf.KafkaTopic, conf.KafkaGroupID, handler)
	defer consumer.Close()

	errCh := make(chan error, 1)

	go func() {
		logger.Printf("Kafka consumer subscribing to topic=%s group=%s", conf.KafkaTopic, conf.KafkaGroupID)
		if err := consumer.Start(ctx); err != nil {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			logger.Printf("consumer error: %v", err)
		}
	}

	logger.Println("notification service stopped")
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
