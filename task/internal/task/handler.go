package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Oniqq60/task_system_control/task/internal/dto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	service   TaskService
	jwtSecret []byte
	rdb       *redis.Client
}

func NewHandler(service TaskService, jwtSecret []byte, rdb *redis.Client) *Handler {
	return &Handler{
		service:   service,
		jwtSecret: jwtSecret,
		rdb:       rdb,
	}
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	adminID, role, err := h.authenticate(r)
	if h.handleAuthError(w, err) {
		return
	}

	if !strings.EqualFold(role, "admin") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req dto.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	// Парсим workerID из строки в uuid.UUID
	workerID, err := uuid.Parse(req.WorkerID)
	if err != nil {
		http.Error(w, "invalid worker_id", http.StatusBadRequest)
		return
	}

	// Вызываем service (здесь workerID и createdBy уже типа uuid.UUID)
	task, err := h.service.CreateTask(r.Context(), req.Message, workerID, adminID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Конвертируем Task в TaskResponse
	resp := dto.TaskResponse{
		ID:        task.ID.String(),
		Message:   task.Message,
		Status:    string(task.Status),
		WorkerID:  task.WorkerID.String(),
		CreatedBy: task.CreatedBy.String(),
		Reason:    task.Reason,
		CreatedAt: task.CreatedAt.Format(time.RFC3339),
		UpdatedAt: task.UpdatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

type authClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

const (
	tokenBlacklistPrefix = "auth:token:blacklist:"
	authCookieName       = "access_token"
)

var errUnauthorized = errors.New("unauthorized")

func (h *Handler) authenticate(r *http.Request) (uuid.UUID, string, error) {
	tokenString, err := h.tokenFromRequest(r)
	if err != nil {
		return uuid.Nil, "", err
	}

	claims := &authClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.jwtSecret, nil
	})
	if err != nil {
		return uuid.Nil, "", errUnauthorized
	}

	if !token.Valid {
		return uuid.Nil, "", errUnauthorized
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, "", errUnauthorized
	}

	if claims.ID == "" {
		return uuid.Nil, "", errUnauthorized
	}

	if h.rdb != nil {
		key := tokenBlacklistPrefix + claims.ID
		exists, err := h.rdb.Exists(r.Context(), key).Result()
		if err != nil {
			return uuid.Nil, "", err
		}
		if exists == 1 {
			return uuid.Nil, "", errUnauthorized
		}
	}

	return userID, claims.Role, nil
}

func (h *Handler) handleAuthError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errUnauthorized) {
		writeUnauthorized(w)
	} else {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
	return true
}

const unauthorizedMessage = "требуется авторизация: войдите в систему"

func writeUnauthorized(w http.ResponseWriter) {
	http.Error(w, unauthorizedMessage, http.StatusUnauthorized)
}

func (h *Handler) tokenFromRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if token, err := extractBearerToken(authHeader); err == nil {
		return token, nil
	}

	if cookie, err := r.Cookie(authCookieName); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token, nil
		}
	}

	return "", errUnauthorized
}

func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", errUnauthorized
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errUnauthorized
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errUnauthorized
	}

	return token, nil
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	_, role, err := h.authenticate(r)
	if h.handleAuthError(w, err) {
		return
	}

	// Получаем ID из path параметра /task/{id}
	// Парсим path: /task/{id}
	path := strings.TrimPrefix(r.URL.Path, "/task/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "id is required in path", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req dto.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if !strings.EqualFold(role, "admin") {
		if req.Message != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if req.Reason != nil {
			if req.Status == nil || !strings.EqualFold(*req.Status, string(StatusNeedsHelp)) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
	}

	// Вызываем service
	task, err := h.service.UpdateTask(r.Context(), id, req.Message, req.Status, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Конвертируем Task в TaskResponse
	resp := dto.TaskResponse{
		ID:        task.ID.String(),
		Message:   task.Message,
		Status:    string(task.Status),
		WorkerID:  task.WorkerID.String(),
		CreatedBy: task.CreatedBy.String(),
		Reason:    task.Reason,
		CreatedAt: task.CreatedAt.Format(time.RFC3339),
		UpdatedAt: task.UpdatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) TaskList(w http.ResponseWriter, r *http.Request) {
	if _, _, err := h.authenticate(r); h.handleAuthError(w, err) {
		return
	}

	// Парсим query параметры
	var workerID *uuid.UUID
	var createdBy *uuid.UUID
	var status *string

	if workerIDStr := r.URL.Query().Get("worker_id"); workerIDStr != "" {
		id, err := uuid.Parse(workerIDStr)
		if err != nil {
			http.Error(w, "invalid worker_id", http.StatusBadRequest)
			return
		}
		workerID = &id
	}

	if createdByStr := r.URL.Query().Get("created_by"); createdByStr != "" {
		id, err := uuid.Parse(createdByStr)
		if err != nil {
			http.Error(w, "invalid created_by", http.StatusBadRequest)
			return
		}
		createdBy = &id
	}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status = &statusStr
	}

	// Вызываем service
	tasks, err := h.service.TaskList(r.Context(), workerID, createdBy, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Конвертируем список задач в TaskResponse
	resp := make([]dto.TaskResponse, 0, len(tasks))
	for _, task := range tasks {
		resp = append(resp, dto.TaskResponse{
			ID:        task.ID.String(),
			Message:   task.Message,
			Status:    string(task.Status),
			WorkerID:  task.WorkerID.String(),
			CreatedBy: task.CreatedBy.String(),
			Reason:    task.Reason,
			CreatedAt: task.CreatedAt.Format(time.RFC3339),
			UpdatedAt: task.UpdatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
