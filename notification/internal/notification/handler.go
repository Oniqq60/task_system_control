package notification

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
)

var (
	ErrEmptyUserID = errors.New("userID is required")
	ErrEmptyTaskID = errors.New("taskID is required")
)

// ManagerResolver возвращает ID менеджера/админа для сотрудника.
type ManagerResolver interface {
	ResolveManager(ctx context.Context, userID string) (string, error)
}

type eventHandler struct {
	notifier Notifier
	resolver ManagerResolver
}

func NewEventHandler(notifier Notifier, resolver ManagerResolver) EventHandler {
	return &eventHandler{
		notifier: notifier,
		resolver: resolver,
	}
}

func (h *eventHandler) HandleEvent(ctx context.Context, event TaskEvent) error {
	if strings.TrimSpace(event.TaskID) == "" {
		return ErrEmptyTaskID
	}
	if strings.TrimSpace(event.UserID) == "" {
		return ErrEmptyUserID
	}
	if !event.NeedsAttention() {
		return nil
	}

	managerID, err := h.resolver.ResolveManager(ctx, event.UserID)
	if err != nil {
		return fmt.Errorf("resolve manager: %w", err)
	}
	if managerID == "" {
		log.Printf("manager not found for user %s, skip notification", event.UserID)
		return nil
	}

	notification := NewNotificationFromEvent(event)
	notification.RecipientID = managerID

	if err := h.notifier.SendNotification(ctx, notification); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}

	return nil
}
