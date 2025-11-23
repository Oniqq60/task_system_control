package services

import (
	"context"
	"fmt"
	"time"

	taskpb "github.com/Oniqq60/task_system_control/gen/proto/task"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TaskService struct {
	conn   *grpc.ClientConn
	client taskpb.TaskServiceClient
}

func NewTaskService(target string) (*TaskService, error) {
	if target == "" {
		return nil, fmt.Errorf("task grpc target is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial task service: %w", err)
	}

	return &TaskService{
		conn:   conn,
		client: taskpb.NewTaskServiceClient(conn),
	}, nil
}

func (s *TaskService) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

func (s *TaskService) CreateTask(ctx context.Context, req *taskpb.CreateTaskRequest) (*taskpb.CreateTaskResponse, error) {
	return s.client.CreateTask(ctx, req)
}

func (s *TaskService) UpdateTask(ctx context.Context, req *taskpb.UpdateTaskRequest) (*taskpb.UpdateTaskResponse, error) {
	return s.client.UpdateTask(ctx, req)
}

func (s *TaskService) ListTasks(ctx context.Context, req *taskpb.TaskListRequest) (*taskpb.TaskListResponse, error) {
	return s.client.TaskList(ctx, req)
}
