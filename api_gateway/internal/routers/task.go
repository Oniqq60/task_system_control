package routers

import (
	"context"
	"net/http"

	"github.com/Oniqq60/task_system_control/api_gateway/internal/services"
	"github.com/Oniqq60/task_system_control/api_gateway/internal/utils"
	taskpb "github.com/Oniqq60/task_system_control/gen/proto/task"
)

type TaskRoutes struct {
	service  *services.TaskService
	verifier *utils.Verifier
}

func NewTaskRoutes(svc *services.TaskService, verifier *utils.Verifier) *TaskRoutes {
	return &TaskRoutes{
		service:  svc,
		verifier: verifier,
	}
}

func (r *TaskRoutes) RegisterHandlers(_ context.Context, mux *http.ServeMux) {
	mux.HandleFunc("POST /task", r.handleCreate)
	mux.HandleFunc("PATCH /task/{id}", r.handleUpdate)
	mux.HandleFunc("GET /task", r.handleList)
}

func (r *TaskRoutes) handleCreate(w http.ResponseWriter, req *http.Request) {
	claims, err := r.authorize(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var payload struct {
		Message  string `json:"message"`
		WorkerID string `json:"worker_id"`
	}
	if err := decodeJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := r.service.CreateTask(req.Context(), &taskpb.CreateTaskRequest{
		Message:   payload.Message,
		WorkerId:  payload.WorkerID,
		CreatedBy: claims.UserID,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (r *TaskRoutes) handleUpdate(w http.ResponseWriter, req *http.Request) {
	if _, err := r.authorize(req); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	taskID := req.PathValue("id")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	var payload struct {
		Message string `json:"message"`
		Status  string `json:"status"`
		Reason  string `json:"reason"`
	}
	if err := decodeJSON(req, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := r.service.UpdateTask(req.Context(), &taskpb.UpdateTaskRequest{
		Id:      taskID,
		Message: payload.Message,
		Status:  payload.Status,
		Reason:  payload.Reason,
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (r *TaskRoutes) handleList(w http.ResponseWriter, req *http.Request) {
	if _, err := r.authorize(req); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	q := req.URL.Query()
	resp, err := r.service.ListTasks(req.Context(), &taskpb.TaskListRequest{
		WorkerId:  q.Get("worker_id"),
		CreatedBy: q.Get("created_by"),
		Status:    q.Get("status"),
	})
	if err != nil {
		handleRPCError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (r *TaskRoutes) authorize(req *http.Request) (utils.Claims, error) {
	token, err := bearerToken(req)
	if err != nil {
		return utils.Claims{}, err
	}
	return r.verifier.ParseToken(req.Context(), token)
}
