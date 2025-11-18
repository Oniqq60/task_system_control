package task

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type TaskService interface {
	CreateTask(ctx context.Context, message string, workerID, createdBy uuid.UUID) (Task, error)
	UpdateTask(ctx context.Context, id uuid.UUID, message, status, reason *string) (Task, error)
	TaskList(ctx context.Context, workerID, createdBy *uuid.UUID, status *string) ([]Task, error)
}

type taskService struct {
	repo          TaskRepository
	kafkaProducer KafkaProducer
}

func NewTaskService(repo TaskRepository, kafkaProducer KafkaProducer) TaskService {
	return &taskService{
		repo:          repo,
		kafkaProducer: kafkaProducer,
	}
}

func (s *taskService) CreateTask(ctx context.Context, message string, workerID, createdBy uuid.UUID) (Task, error) {
	if message == "" {
		return Task{}, errors.New("message is required")
	}

	task := Task{
		ID:        uuid.New(),
		Message:   message,
		Status:    StatusInProgress,
		WorkerID:  workerID,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return Task{}, err
	}

	return task, nil
}

func (s *taskService) UpdateTask(ctx context.Context, id uuid.UUID, message, status, reason *string) (Task, error) {
	// Проверяем существование задачи
	existingTask, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return Task{}, err
	}

	updates := Task{
		UpdatedAt: time.Now(),
	}

	// Обновляем message, если передан
	if message != nil {
		if *message == "" {
			return Task{}, errors.New("message cannot be empty")
		}
		updates.Message = *message
	}

	// Обновляем status, если передан
	if status != nil {
		validStatus := Status(*status)
		if validStatus != StatusInProgress && validStatus != StatusCompleted && validStatus != StatusNeedsHelp {
			return Task{}, errors.New("invalid status: must be IN_PROGRESS, COMPLETED, or NEEDS_HELP")
		}
		updates.Status = validStatus

		// Если статус NEEDS_HELP, требуется reason
		if validStatus == StatusNeedsHelp {
			if reason == nil || *reason == "" {
				return Task{}, errors.New("reason is required when status is NEEDS_HELP")
			}
			updates.Reason = reason
		} else {
			// Для других статусов reason очищаем
			updates.Reason = nil
		}
	} else if reason != nil {
		// Если передан reason, но статус не меняется
		// reason можно обновить только если текущий статус NEEDS_HELP
		if existingTask.Status == StatusNeedsHelp {
			updates.Reason = reason
		}
		// Если текущий статус не NEEDS_HELP, игнорируем reason (не обновляем)
	}

	// Обновляем задачу
	if err := s.repo.UpdateTask(ctx, id, updates); err != nil {
		return Task{}, err
	}

	// Получаем обновлённую задачу
	task, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return Task{}, err
	}

	// Если статус изменился на NEEDS_HELP, отправляем событие в Kafka
	if status != nil && Status(*status) == StatusNeedsHelp && existingTask.Status != StatusNeedsHelp {
		if s.kafkaProducer != nil {
			event := TaskEvent{
				TaskID:    task.ID.String(),
				UserID:    task.WorkerID.String(),
				Status:    string(StatusNeedsHelp),
				Reason:    *reason,
				Timestamp: time.Now(),
			}
			// Отправляем событие асинхронно (не блокируем ответ)
			go func() {
				if err := s.kafkaProducer.SendTaskEvent(context.Background(), event); err != nil {
					// Логируем ошибку (в production лучше использовать structured logging)
					// log.Printf("failed to send task event to kafka: %v", err)
				}
			}()
		}
	}

	return task, nil
}

func (s *taskService) TaskList(ctx context.Context, workerID, createdBy *uuid.UUID, status *string) ([]Task, error) {
	// Валидация status, если передан
	if status != nil {
		validStatus := Status(*status)
		if validStatus != StatusInProgress && validStatus != StatusCompleted && validStatus != StatusNeedsHelp {
			return nil, errors.New("invalid status: must be IN_PROGRESS, COMPLETED, or NEEDS_HELP")
		}
	}

	return s.repo.TaskList(ctx, workerID, createdBy, status)
}
