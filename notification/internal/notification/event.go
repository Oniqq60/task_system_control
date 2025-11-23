package notification

import (
	"fmt"
	"strings"
	"time"
)

// TaskEvent представляет событие изменения задачи из Kafka.
// Структура должна совпадать с task/internal/task/kafka.go
type TaskEvent struct {
	TaskID    string    `json:"taskId"`
	UserID    string    `json:"userId"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Notification описывает уведомление, которое будет отправлено.
type Notification struct {
	Type        string
	TaskID      string
	UserID      string
	Message     string
	RecipientID string
	CreatedAt   time.Time
}

// NeedsAttention возвращает true, если событие требует уведомления администратора.
func (e TaskEvent) NeedsAttention() bool {
	return strings.EqualFold(e.Status, "NEEDS_HELP")
}

// NewNotificationFromEvent создаёт Notification из TaskEvent.
func NewNotificationFromEvent(event TaskEvent) Notification {
	message := fmt.Sprintf(
		"Задача %s требует помощи. Причина: %s",
		event.TaskID,
		event.Reason,
	)

	return Notification{
		Type:      "task_needs_help",
		TaskID:    event.TaskID,
		UserID:    event.UserID,
		Message:   message,
		CreatedAt: time.Now(),
	}
}
