package document

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Oniqq60/task_system_control/document/internal/dto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	service   Service
	jwtSecret []byte
	maxSize   int64
	rdb       *redis.Client
}

func NewHandler(service Service, jwtSecret []byte, maxSize int64, rdb *redis.Client) *Handler {
	return &Handler{
		service:   service,
		jwtSecret: jwtSecret,
		maxSize:   maxSize,
		rdb:       rdb,
	}
}

func (h *Handler) AddDocument(w http.ResponseWriter, r *http.Request) {
	requester, err := h.ensureAuthorized(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	// Парсим multipart/form-data
	if err := r.ParseMultipartForm(h.maxSize); err != nil {
		if err == http.ErrNotMultipart {
			// Fallback: попробуем JSON для обратной совместимости
			var req dto.AddDocumentRequest
			if jsonErr := json.NewDecoder(r.Body).Decode(&req); jsonErr != nil {
				http.Error(w, "invalid request format: expected multipart/form-data or json", http.StatusBadRequest)
				return
			}
			// Обработка JSON формата (legacy)
			if requester.Role != RoleAdmin && requester.UserID != req.OwnerID {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			// Для JSON формата нужна base64 декодировка (старый способ)
			http.Error(w, "json format with base64 is deprecated, use multipart/form-data", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Извлекаем файл
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Читаем содержимое файла
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	// Извлекаем остальные поля из form
	taskID := r.FormValue("task_id")
	ownerID := r.FormValue("owner_id")
	filename := r.FormValue("filename")
	contentType := r.FormValue("content_type")
	tagsStr := r.FormValue("tags")

	// Если filename не указан, используем имя загруженного файла
	if filename == "" {
		filename = fileHeader.Filename
	}

	// Санитизация и валидация имени файла
	filename = SanitizeFilename(filename)
	if err := ValidateFilename(filename); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Если content_type не указан, пытаемся определить из файла
	if contentType == "" {
		contentType = fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			// Простое определение по расширению
			contentType = http.DetectContentType(content)
		}
	}

	// Валидация content type
	if err := ValidateContentType(contentType); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Парсим tags (ожидаем JSON массив строк или разделённые запятыми)
	var tags []string
	if tagsStr != "" {
		// Пробуем распарсить как JSON
		if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
			// Если не JSON, разделяем по запятым
			tagList := strings.Split(tagsStr, ",")
			tags = make([]string, 0, len(tagList))
			for _, tag := range tagList {
				if trimmed := strings.TrimSpace(tag); trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
		}
	}

	// Валидация обязательных полей
	if taskID == "" {
		http.Error(w, "task_id is required", http.StatusBadRequest)
		return
	}
	if ownerID == "" {
		http.Error(w, "owner_id is required", http.StatusBadRequest)
		return
	}

	// Проверка прав доступа
	if requester.Role != RoleAdmin && requester.UserID != ownerID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	metadata, _, err := h.service.AddDocument(r.Context(), AddDocumentInput{
		TaskID:      taskID,
		OwnerID:     ownerID,
		Filename:    filename,
		ContentType: contentType,
		Tags:        tags,
		Content:     content,
		MaxSize:     h.maxSize,
	})
	if err != nil {
		if errors.Is(err, ErrFileTooLarge) {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		if errors.Is(err, ErrInvalidTaskID) || errors.Is(err, ErrInvalidOwnerID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := mapMetadata(metadata)
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	requester, err := h.ensureAuthorized(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/documents/")
	if id == "" {
		http.Error(w, "document id required", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteDocument(r.Context(), id, requester); err != nil {
		if errors.Is(err, ErrForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if errors.Is(err, ErrInvalidDocumentID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	requester, err := h.ensureAuthorized(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/documents/")
	if id == "" {
		http.Error(w, "document id required", http.StatusBadRequest)
		return
	}

	metadata, content, err := h.service.GetDocument(r.Context(), id, requester)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if errors.Is(err, ErrInvalidDocumentID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Возвращаем бинарный файл с правильными заголовками
	// Экранируем filename для защиты от XSS
	safeFilename := EscapeFilename(metadata.Filename)
	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+safeFilename+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(content)), 10))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) ListByTask(w http.ResponseWriter, r *http.Request) {
	requester, err := h.ensureAuthorized(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	taskID := strings.TrimPrefix(r.URL.Path, "/documents/task/")
	if taskID == "" {
		http.Error(w, "task id required", http.StatusBadRequest)
		return
	}

	docs, err := h.service.GetDocumentsByTask(r.Context(), taskID, requester)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if errors.Is(err, ErrInvalidTaskID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		resp = append(resp, mapMetadata(doc))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListByOwner(w http.ResponseWriter, r *http.Request) {
	requester, err := h.ensureAuthorized(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	ownerID := strings.TrimPrefix(r.URL.Path, "/documents/owner/")
	if ownerID == "" {
		http.Error(w, "owner id required", http.StatusBadRequest)
		return
	}

	docs, err := h.service.GetDocumentsByOwner(r.Context(), ownerID, requester)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if errors.Is(err, ErrInvalidOwnerID) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		resp = append(resp, mapMetadata(doc))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ensureAuthorized(r *http.Request) (Requester, error) {
	tokenString, err := h.tokenFromRequest(r)
	if err != nil {
		return Requester{}, err
	}

	claims := &authClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return Requester{}, errUnauthorized
	}

	if claims.ID == "" || claims.UserID == "" {
		return Requester{}, errUnauthorized
	}

	if h.rdb != nil {
		key := tokenBlacklistPrefix + claims.ID
		exists, redisErr := h.rdb.Exists(r.Context(), key).Result()
		if redisErr != nil {
			return Requester{}, redisErr
		}
		if exists > 0 {
			return Requester{}, errUnauthorized
		}
	}

	return Requester{UserID: claims.UserID, Role: Role(strings.ToLower(claims.Role))}, nil
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

func writeAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, errUnauthorized) {
		http.Error(w, "требуется авторизация: войдите в систему", http.StatusUnauthorized)
		return
	}
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func mapMetadata(metadata Metadata) map[string]interface{} {
	return map[string]interface{}{
		"id":           metadata.ID.Hex(),
		"task_id":      metadata.TaskID,
		"owner_id":     metadata.OwnerID,
		"filename":     metadata.Filename,
		"content_type": metadata.ContentType,
		"size":         metadata.Size,
		"tags":         metadata.Tags,
		"uploaded_at":  metadata.UploadedAt.Unix(),
		"checksum":     metadata.Checksum,
		"minio_object": metadata.MinioObject,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/documents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.AddDocument(w, r)
	})
	mux.HandleFunc("/documents/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/documents/task/"):
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.ListByTask(w, r)
		case strings.HasPrefix(r.URL.Path, "/documents/owner/"):
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.ListByOwner(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				h.GetDocument(w, r)
			case http.MethodDelete:
				h.DeleteDocument(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}
	})
	return mux
}

func (h *Handler) tokenFromRequest(r *http.Request) (string, error) {
	if token, err := extractBearerToken(r.Header.Get("Authorization")); err == nil {
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
