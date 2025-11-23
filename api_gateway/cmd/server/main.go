package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	appcfg "github.com/Oniqq60/task_system_control/api_gateway/internal/config"
	"github.com/Oniqq60/task_system_control/api_gateway/internal/middleware"
	"github.com/Oniqq60/task_system_control/api_gateway/internal/routers"
	"github.com/Oniqq60/task_system_control/api_gateway/internal/services"
	"github.com/Oniqq60/task_system_control/api_gateway/internal/utils"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("api gateway stopped: %v", err)
	}
}

func run() error {
	cfg, err := appcfg.Load()
	if err != nil {
		return err
	}

	logger := log.New(os.Stdout, "[API_GATEWAY] ", log.LstdFlags|log.Lshortfile)

	jwtVerifier, err := utils.NewVerifier(cfg.JWTSecret)
	if err != nil {
		return err
	}

	authSvc, err := services.NewAuthService(cfg.AuthGRPCAddr)
	if err != nil {
		return err
	}
	defer authSvc.Close()

	taskSvc, err := services.NewTaskService(cfg.TaskGRPCAddr)
	if err != nil {
		return err
	}
	defer taskSvc.Close()

	documentSvc, err := services.NewDocumentService(cfg.DocumentGRPCAddr)
	if err != nil {
		return err
	}
	defer documentSvc.Close()

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
	cors := middleware.NewCORS(middleware.CORSOptions{
		AllowedOrigins:   cfg.AllowedCORSOrigins,
		AllowCredentials: true,
	})

	router, err := routers.New(routers.Dependencies{
		Auth:                 authSvc,
		Task:                 taskSvc,
		Document:             documentSvc,
		JWTVerifier:          jwtVerifier,
		Middleware:           []func(http.Handler) http.Handler{rateLimiter.Middleware, cors},
		MaxDocumentBodyBytes: cfg.ForwardResponseLimit,
	})
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Printf("HTTP server listening on :%s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownGracePeriod)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
