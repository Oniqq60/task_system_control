package notification

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Notifier отвечает за доставку уведомлений
type Notifier interface {
	SendNotification(ctx context.Context, notification Notification) error
}

type logNotifier struct {
	logger *log.Logger
}

func NewLogNotifier(logger *log.Logger) Notifier {
	if logger == nil {
		logger = log.Default()
	}
	return &logNotifier{logger: logger}
}

func (n *logNotifier) SendNotification(ctx context.Context, notification Notification) error {
	entry := fmt.Sprintf(
		"[NOTIFICATION] type=%s task=%s user=%s recipient=%s message=%q at=%s",
		notification.Type,
		notification.TaskID,
		notification.UserID,
		notification.RecipientID,
		notification.Message,
		notification.CreatedAt.Format(time.RFC3339),
	)
	n.logger.Println(entry)
	return nil
}
