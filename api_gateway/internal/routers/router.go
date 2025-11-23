package routers

import (
	"context"
	"errors"
	"net/http"

	"github.com/Oniqq60/task_system_control/api_gateway/internal/services"
	"github.com/Oniqq60/task_system_control/api_gateway/internal/utils"
)

type Dependencies struct {
	Auth                 *services.AuthService
	Task                 *services.TaskService
	Document             *services.DocumentService
	JWTVerifier          *utils.Verifier
	Middleware           []func(http.Handler) http.Handler
	MaxDocumentBodyBytes int64
}

type Router struct {
	mux     *http.ServeMux
	handler http.Handler
}

func New(deps Dependencies) (*Router, error) {
	if deps.Auth == nil || deps.Task == nil || deps.Document == nil {
		return nil, errors.New("all downstream services must be provided")
	}
	if deps.JWTVerifier == nil {
		return nil, errors.New("jwt verifier is required")
	}

	mux := http.NewServeMux()
	ctx := context.Background()

	NewAuthRoutes(deps.Auth).RegisterHandlers(ctx, mux)
	NewTaskRoutes(deps.Task, deps.JWTVerifier).RegisterHandlers(ctx, mux)
	NewDocumentRoutes(deps.Document, deps.JWTVerifier, deps.MaxDocumentBodyBytes).RegisterHandlers(ctx, mux)

	var handler http.Handler = mux
	for i := len(deps.Middleware) - 1; i >= 0; i-- {
		handler = deps.Middleware[i](handler)
	}

	return &Router{
		mux:     mux,
		handler: handler,
	}, nil
}

func (r *Router) Handler() http.Handler {
	if r == nil {
		return nil
	}
	return r.handler
}
