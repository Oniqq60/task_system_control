package routers

import (
	"context"
	"net/http"

	"github.com/Oniqq60/task_system_control/api_gateway/internal/services"
	authpb "github.com/Oniqq60/task_system_control/gen/proto/auth/v1"
)

type AuthRoutes struct {
	service *services.AuthService
}

func NewAuthRoutes(svc *services.AuthService) *AuthRoutes {
	return &AuthRoutes{service: svc}
}

func (r *AuthRoutes) RegisterHandlers(_ context.Context, mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", r.handleRegister)
	mux.HandleFunc("POST /auth/login", r.handleLogin)
	mux.HandleFunc("POST /auth/validate", r.handleValidate)
}

func (r *AuthRoutes) handleRegister(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Name      string `json:"name"`
		Email     string `json:"email"`
		Password  string `json:"password"`
		Role      string `json:"role"`
		ManagerID string `json:"manager_id"`
	}
	if err := decodeJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := r.service.Register(req.Context(), &authpb.RegisterRequest{
		Name:      payload.Name,
		Email:     payload.Email,
		Password:  payload.Password,
		Role:      payload.Role,
		ManagerId: payload.ManagerID,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (r *AuthRoutes) handleLogin(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := r.service.Login(req.Context(), &authpb.LoginRequest{
		Email:    payload.Email,
		Password: payload.Password,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (r *AuthRoutes) handleValidate(w http.ResponseWriter, req *http.Request) {
	var payload struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(req, &payload); err != nil && err != errEmptyBody {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	token := payload.Token
	if token == "" {
		bearer, err := bearerToken(req)
		if err != nil {
			writeError(w, http.StatusBadRequest, "token is required")
			return
		}
		token = bearer
	}

	resp, err := r.service.ValidateToken(req.Context(), &authpb.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
