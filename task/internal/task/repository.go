package task

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskRepository interface {
	CreateTask(ctx context.Context, t Task) error
	UpdateTask(ctx context.Context, id uuid.UUID, updates Task) error
	GetTask(ctx context.Context, id uuid.UUID) (Task, error)
	TaskList(ctx context.Context, workerID, createdBy *uuid.UUID, status *string) ([]Task, error)
}

type taskRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) CreateTask(ctx context.Context, t Task) error {
	return r.db.WithContext(ctx).Create(&t).Error
}

func (r *taskRepository) UpdateTask(ctx context.Context, id uuid.UUID, updates Task) error {
	// Используем map для обновления, чтобы иметь точный контроль над обновляемыми полями
	updateMap := make(map[string]interface{})

	// Всегда обновляем updated_at
	updateMap["updated_at"] = updates.UpdatedAt

	// Обновляем message, если оно не пустое
	if updates.Message != "" {
		updateMap["message"] = updates.Message
	}

	// Обновляем status, если оно не пустое
	if updates.Status != "" {
		updateMap["status"] = updates.Status
	}

	// Для reason: service слой уже обработал всю логику
	// Если status меняется, service установил правильное значение reason (или nil для очистки)
	// Если только reason обновляется (status не меняется), service установил updates.Reason
	// Проверяем оба случая:
	// 1. Если status меняется - обновляем reason (может быть nil для очистки)
	// 2. Если status не меняется, но reason != nil - обновляем reason
	if updates.Status != "" {
		// Статус меняется - reason уже обработан в service (может быть nil для очистки)
		if updates.Reason != nil {
			updateMap["reason"] = *updates.Reason
		} else {
			// Service установил nil для очистки reason при смене статуса
			updateMap["reason"] = nil
		}
	} else if updates.Reason != nil {
		// Статус не меняется, но reason обновляется
		updateMap["reason"] = *updates.Reason
	}

	return r.db.WithContext(ctx).Model(&Task{}).Where("id = ?", id).Updates(updateMap).Error
}

func (r *taskRepository) GetTask(ctx context.Context, id uuid.UUID) (Task, error) {
	var task Task
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error
	return task, err
}

func (r *taskRepository) TaskList(ctx context.Context, workerID, createdBy *uuid.UUID, status *string) ([]Task, error) {
	var tasks []Task
	tx := r.db.WithContext(ctx)

	if workerID != nil {
		tx = tx.Where("worker_id = ?", *workerID)
	}
	if createdBy != nil {
		tx = tx.Where("created_by = ?", *createdBy)
	}
	if status != nil {
		tx = tx.Where("status = ?", *status)
	}

	if err := tx.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}
