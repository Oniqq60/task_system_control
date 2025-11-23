package routers

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Oniqq60/task_system_control/api_gateway/internal/services"
	"github.com/Oniqq60/task_system_control/api_gateway/internal/utils"
	documentpb "github.com/Oniqq60/task_system_control/gen/proto/document"
)

type DocumentRoutes struct {
	service      *services.DocumentService
	verifier     *utils.Verifier
	maxBodyBytes int64
}

func NewDocumentRoutes(svc *services.DocumentService, verifier *utils.Verifier, maxBodyBytes int64) *DocumentRoutes {
	return &DocumentRoutes{
		service:      svc,
		verifier:     verifier,
		maxBodyBytes: maxBodyBytes,
	}
}

func (r *DocumentRoutes) RegisterHandlers(_ context.Context, mux *http.ServeMux) {
	mux.HandleFunc("POST /document", r.handleAdd)
	mux.HandleFunc("DELETE /document/{id}", r.handleDelete)
	mux.HandleFunc("GET /document/{id}", r.handleGet)
	mux.HandleFunc("GET /document/task/{taskId}", r.handleByTask)
	mux.HandleFunc("GET /document/owner", r.handleByOwner)
}

func (r *DocumentRoutes) handleAdd(w http.ResponseWriter, req *http.Request) {
	claims, err := r.authorize(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Проверяем Content-Type
	contentType := req.Header.Get("Content-Type")

	var payload struct {
		TaskID      string   `json:"task_id"`
		Filename    string   `json:"filename"`
		ContentType string   `json:"content_type"`
		FileBase64  string   `json:"file_base64"`
		Tags        []string `json:"tags"`
	}
	var fileContent []byte

	// Сценарий 1: multipart/form-data
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := req.ParseMultipartForm(r.maxBodyBytes); err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "file too large or invalid multipart")
			return
		}

		taskID := req.FormValue("task_id")
		filename := req.FormValue("filename")
		contentTypeValue := req.FormValue("content_type")
		tags := req.FormValue("tags")

		file, _, err := req.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "missing file field")
			return
		}
		defer file.Close()

		// Читаем содержимое
		fileContent, err = io.ReadAll(io.LimitReader(file, r.maxBodyBytes))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read file")
			return
		}

		// Заполняем payload
		payload.TaskID = taskID
		payload.Filename = filename
		payload.ContentType = contentTypeValue
		if tags != "" {
			payload.Tags = strings.Split(tags, ",")
		}

	} else {
		// Сценарий 2: application/json
		if err := decodeJSON(req, &payload); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		data, err := base64.StdEncoding.DecodeString(payload.FileBase64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "file_base64 must be base64 encoded")
			return
		}
		fileContent = data
	}

	// Валидация
	if payload.TaskID == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}
	if payload.Filename == "" {
		writeError(w, http.StatusBadRequest, "filename is required")
		return
	}
	if payload.ContentType == "" {
		writeError(w, http.StatusBadRequest, "content_type is required")
		return
	}
	if len(fileContent) == 0 {
		writeError(w, http.StatusBadRequest, "file content is empty")
		return
	}
	if len(fileContent) > int(r.maxBodyBytes) {
		writeError(w, http.StatusRequestEntityTooLarge, "file too large")
		return
	}

	// Получаем токен из запроса для передачи в gRPC
	token, _ := bearerToken(req)
	ctx := context.WithValue(req.Context(), "jwt_token", token)

	// Вызов gRPC
	resp, err := r.service.AddDocument(ctx, &documentpb.AddDocumentRequest{
		TaskId:      payload.TaskID,
		OwnerId:     claims.UserID,
		FileContent: fileContent,
		Filename:    payload.Filename,
		ContentType: payload.ContentType,
		Tags:        payload.Tags,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (r *DocumentRoutes) handleDelete(w http.ResponseWriter, req *http.Request) {
	claims, err := r.authorize(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	docID := req.PathValue("id")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "document id is required")
		return
	}

	// Получаем токен из запроса для передачи в gRPC
	token, _ := bearerToken(req)
	ctx := context.WithValue(req.Context(), "jwt_token", token)

	resp, err := r.service.DeleteDocument(ctx, &documentpb.DeleteDocumentRequest{
		Id:      docID,
		OwnerId: claims.UserID,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (r *DocumentRoutes) handleGet(w http.ResponseWriter, req *http.Request) {
	if _, err := r.authorize(req); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	docID := req.PathValue("id")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "document id is required")
		return
	}

	// Получаем токен из запроса для передачи в gRPC
	token, _ := bearerToken(req)
	ctx := context.WithValue(req.Context(), "jwt_token", token)

	resp, err := r.service.GetDocument(ctx, &documentpb.GetDocumentRequest{
		Id: docID,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	fileContent := resp.GetFileContent()
	if r.maxBodyBytes > 0 && len(fileContent) > int(r.maxBodyBytes) {
		writeError(w, http.StatusRequestEntityTooLarge, "document exceeds response limits")
		return
	}

	// Устанавливаем правильные HTTP заголовки для возврата файла
	contentType := resp.GetContentType()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileContent)))
	
	// Content-Disposition для скачивания файла с правильным именем
	filename := resp.GetFilename()
	if filename == "" {
		filename = "document"
	}
	// Экранируем имя файла для безопасного использования в заголовке
	escapedFilename := escapeFilenameForHeader(filename)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, escapedFilename, escapeFilenameForHeaderUTF8(filename)))
	
	// Возвращаем бинарные данные напрямую
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(fileContent); err != nil {
		// Логируем ошибку, но ответ уже отправлен
		return
	}
}

func (r *DocumentRoutes) handleByTask(w http.ResponseWriter, req *http.Request) {
	if _, err := r.authorize(req); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	taskID := req.PathValue("taskId")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	// Получаем токен из запроса для передачи в gRPC
	token, _ := bearerToken(req)
	ctx := context.WithValue(req.Context(), "jwt_token", token)

	resp, err := r.service.GetDocumentsByTask(ctx, &documentpb.GetDocumentsByTaskRequest{
		TaskId: taskID,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (r *DocumentRoutes) handleByOwner(w http.ResponseWriter, req *http.Request) {
	claims, err := r.authorize(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Получаем токен из запроса для передачи в gRPC
	token, _ := bearerToken(req)
	ctx := context.WithValue(req.Context(), "jwt_token", token)

	resp, err := r.service.GetDocumentsByOwner(ctx, &documentpb.GetDocumentsByOwnerRequest{
		OwnerId: claims.UserID,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (r *DocumentRoutes) authorize(req *http.Request) (utils.Claims, error) {
	token, err := bearerToken(req)
	if err != nil {
		return utils.Claims{}, err
	}
	return r.verifier.ParseToken(req.Context(), token)
}

// escapeFilenameForHeader экранирует имя файла для использования в Content-Disposition header
func escapeFilenameForHeader(filename string) string {
	// Экранируем кавычки и обратные слеши
	filename = strings.ReplaceAll(filename, `"`, `\"`)
	filename = strings.ReplaceAll(filename, `\`, `\\`)
	return filename
}

// escapeFilenameForHeaderUTF8 экранирует имя файла для RFC 5987 (filename*)
func escapeFilenameForHeaderUTF8(filename string) string {
	return url.QueryEscape(filename)
}
