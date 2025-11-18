package task

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Oniqq60/task_system_control/gen/proto/task"
	"github.com/google/uuid"
)

// GrpcHandler реализует gRPC сервер для Task Service
type GrpcHandler struct {
	pb.UnimplementedTaskServiceServer
	service TaskService
}

// NewGrpcHandler создаёт новый gRPC handler
func NewGrpcHandler(service TaskService) *GrpcHandler {
	return &GrpcHandler{
		service: service,
	}
}

// CreateTask создаёт новую задачу
func (h *GrpcHandler) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	if req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	// Парсим workerID из строки в uuid.UUID
	workerID, err := uuid.Parse(req.WorkerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid worker_id")
	}

	// Парсим createdBy из строки в uuid.UUID
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	// Вызываем service
	task, err := h.service.CreateTask(ctx, req.Message, workerID, createdBy)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Конвертируем Task в protobuf CreateTaskResponse
	return &pb.CreateTaskResponse{
		Id:        task.ID.String(),
		Message:   task.Message,
		Status:    string(task.Status),
		WorkerId:  task.WorkerID.String(),
		CreatedBy: task.CreatedBy.String(),
	}, nil
}

// UpdateTask обновляет задачу
func (h *GrpcHandler) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Парсим id из строки в uuid.UUID
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	// Конвертируем опциональные поля
	var message *string
	if req.Message != "" {
		message = &req.Message
	}

	var statusStr *string
	if req.Status != "" {
		statusStr = &req.Status
	}

	var reason *string
	if req.Reason != "" {
		reason = &req.Reason
	}

	// Вызываем service
	task, err := h.service.UpdateTask(ctx, id, message, statusStr, reason)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Конвертируем Task в protobuf UpdateTaskResponse
	resp := &pb.UpdateTaskResponse{
		Id:       task.ID.String(),
		Message:  task.Message,
		Status:   string(task.Status),
		WorkerId: task.WorkerID.String(),
	}
	if task.Reason != nil {
		resp.Reason = *task.Reason
	}

	return resp, nil
}

// TaskList возвращает список задач
func (h *GrpcHandler) TaskList(ctx context.Context, req *pb.TaskListRequest) (*pb.TaskListResponse, error) {
	// Конвертируем фильтры
	var workerID *uuid.UUID
	if req.WorkerId != "" {
		id, err := uuid.Parse(req.WorkerId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid worker_id")
		}
		workerID = &id
	}

	var createdBy *uuid.UUID
	if req.CreatedBy != "" {
		id, err := uuid.Parse(req.CreatedBy)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid created_by")
		}
		createdBy = &id
	}

	var statusStr *string
	if req.Status != "" {
		statusStr = &req.Status
	}

	// Вызываем service
	tasks, err := h.service.TaskList(ctx, workerID, createdBy, statusStr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Конвертируем список задач в protobuf Task
	pbTasks := make([]*pb.Task, 0, len(tasks))
	for _, task := range tasks {
		pbTask := &pb.Task{
			Id:        task.ID.String(),
			Message:   task.Message,
			Status:    string(task.Status),
			WorkerId:  task.WorkerID.String(),
			CreatedBy: task.CreatedBy.String(),
			CreatedAt: task.CreatedAt.Unix(),
			UpdatedAt: task.UpdatedAt.Unix(),
		}
		if task.Reason != nil {
			pbTask.Reason = *task.Reason
		}
		pbTasks = append(pbTasks, pbTask)
	}

	return &pb.TaskListResponse{
		Tasks: pbTasks,
	}, nil
}
