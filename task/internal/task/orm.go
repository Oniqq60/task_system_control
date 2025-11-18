package task

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusInProgress Status = "IN_PROGRESS"
	StatusCompleted  Status = "COMPLETED"
	StatusNeedsHelp  Status = "NEEDS_HELP"
)

type Task struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Message   string    `json:"message" gorm:"not null"`
	Status    Status    `json:"status" gorm:"type:text;not null;default:'IN_PROGRESS';check:status IN ('IN_PROGRESS', 'COMPLETED', 'NEEDS_HELP')"`
	WorkerID  uuid.UUID `json:"worker_id" gorm:"type:uuid;not null"`
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	Reason    *string   `json:"reason,omitempty" gorm:"type:text"` // Причина для статуса NEEDS_HELP
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"not null;default:now()"`
}
