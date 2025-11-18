package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Oniqq60/task_system_control/auth/internal/dto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	service   UserService
	jwtSecret []byte
	rdb       *redis.Client
}

// NewHandler создает новый экземпляр Handler
func NewUserHandler(service UserService, jwtSecret []byte, rdb *redis.Client) *Handler {
	return &Handler{
		service:   service,
		jwtSecret: jwtSecret,
		rdb:       rdb,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Валидация email
	if err := ValidateEmail(req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация пароля
	if err := ValidatePassword(req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация имени
	if err := ValidateName(req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// validate/normalize role from request, default to employee
	role := RoleEmployee
	if req.Role != "" {
		switch Role(req.Role) {
		case RoleAdmin:
			role = RoleAdmin
		case RoleEmployee:
			role = RoleEmployee
		default:
			http.Error(w, "invalid role", http.StatusBadRequest)
			return
		}
	}

	// Санитизация входных данных
	u := Users{
		ID:            uuid.New(),
		Name:          SanitizeString(req.Name),
		Email:         strings.ToLower(strings.TrimSpace(req.Email)),
		Password_hash: req.Password, // will be hashed in service
		Role:          role,
		Created_at:    time.Now(),
	}
	if err := h.service.Register(r.Context(), u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := dto.RegisterResponse{ID: u.ID.String(), Email: u.Email, Name: u.Name}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Валидация email
	if err := ValidateEmail(req.Email); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Нормализация email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// rate limit by email+ip (улучшенная версия с учетом прокси)
	ip := getClientIP(r)
	key := "auth:login:attempts:" + req.Email + ":" + ip
	if cnt, err := h.rdb.Get(r.Context(), key).Int64(); err == nil && cnt >= 5 {
		http.Error(w, "too many attempts", http.StatusTooManyRequests)
		return
	}
	user, claims, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		// increment attempts and set TTL on first failure
		val, _ := h.rdb.Incr(r.Context(), key).Result()
		if val == 1 {
			_ = h.rdb.Expire(r.Context(), key, 10*time.Minute).Err()
		}
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	// success: reset counter
	_ = h.rdb.Del(r.Context(), key).Err()
	token, err := SignToken(claims, h.jwtSecret)
	if err != nil {
		http.Error(w, "failed to sign token", http.StatusInternalServerError)
		return
	}
	expiresIn := int64(0)
	if claims.ExpiresAt != nil {
		expiresIn = claims.ExpiresAt.Time.Unix() - time.Now().Unix()
	}
	_ = user
	if err := h.setAuthCookie(w, token, expiresIn); err != nil {
		log.Printf("Login: failed to set auth cookie: %v", err)
	}
	resp := dto.LoginResponse{AccessToken: token, ExpiresIn: expiresIn}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// getClientIP извлекает реальный IP клиента с учетом прокси
func getClientIP(r *http.Request) string {
	// Проверяем заголовки прокси
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For может содержать несколько IP через запятую
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	// Fallback на RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

const tokenBlacklistPrefix = "auth:token:blacklist:"
const authCookieName = "access_token"

var errTokenRequired = errors.New("token required")

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tokenString, err := h.tokenFromRequest(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.ID == "" || claims.ExpiresAt == nil {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	key := tokenBlacklistPrefix + claims.ID
	if err := h.rdb.Set(r.Context(), key, "revoked", ttl).Err(); err != nil {
		http.Error(w, "failed to logout", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tokenString, err := h.tokenFromRequest(r)
	if err != nil {
		log.Printf("Profile: failed to extract token: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := ParseToken(tokenString, h.jwtSecret)
	if err != nil {
		log.Printf("Profile: failed to parse token: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	log.Printf("Profile: parsed token - UserID: %v, Subject: %s, ID: %s", claims.UserID, claims.Subject, claims.ID)

	if claims.ID != "" && h.rdb != nil {
		key := tokenBlacklistPrefix + claims.ID
		if exists, redisErr := h.rdb.Exists(r.Context(), key).Result(); redisErr == nil && exists > 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Если Redis недоступен, просто пропускаем проверку черного списка
		// (логируем предупреждение, но не блокируем запрос)
	}

	if claims.UserID == uuid.Nil {
		log.Printf("Profile: UserID is nil, Subject: %s", claims.Subject)
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}

	user, err := h.service.Profile(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to fetch profile", http.StatusInternalServerError)
		return
	}

	resp := dto.ProfileResponse{
		ID:             user.ID.String(),
		Name:           user.Name,
		Email:          user.Email,
		Role:           string(user.Role),
		CompletedTasks: user.Completed_tasks,
		CreatedAt:      user.Created_at,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) tokenFromRequest(r *http.Request) (string, error) {
	if token, err := extractBearerToken(r.Header.Get("Authorization")); err == nil {
		return token, nil
	}
	if cookie, err := r.Cookie(authCookieName); err == nil {
		token := strings.TrimSpace(cookie.Value)
		if token != "" {
			return token, nil
		}
	}
	return "", errTokenRequired
}

func (h *Handler) setAuthCookie(w http.ResponseWriter, token string, expiresIn int64) error {
	if token == "" {
		return errors.New("empty token")
	}
	// Определяем, используется ли HTTPS (через заголовок или переменную окружения)
	isSecure := os.Getenv("HTTPS_ENABLED") == "true" || os.Getenv("HTTPS_ENABLED") == "1"

	cookie := &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecure, // Используем HTTPS если включен
	}
	if expiresIn > 0 {
		duration := time.Duration(expiresIn) * time.Second
		cookie.Expires = time.Now().Add(duration)
		cookie.MaxAge = int(duration.Seconds())
	}
	http.SetCookie(w, cookie)
	return nil
}

func (h *Handler) clearAuthCookie(w http.ResponseWriter) {
	isSecure := os.Getenv("HTTPS_ENABLED") == "true" || os.Getenv("HTTPS_ENABLED") == "1"
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isSecure,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", errTokenRequired
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errTokenRequired
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errTokenRequired
	}

	return token, nil
}
