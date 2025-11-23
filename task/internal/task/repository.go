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

	updateMap := make(map[string]interface{})

	updateMap["updated_at"] = updates.UpdatedAt

	if updates.Message != "" {
		updateMap["message"] = updates.Message
	}

	if updates.Status != "" {
		updateMap["status"] = updates.Status
	}

	if updates.Status != "" {

		if updates.Reason != nil {
			updateMap["reason"] = *updates.Reason
		} else {

			updateMap["reason"] = nil
		}
	} else if updates.Reason != nil {

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
