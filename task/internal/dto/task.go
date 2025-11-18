package dto

// CreateTaskRequest - запрос на создание задачи (HTTP)
type CreateTaskRequest struct {
	Message  string `json:"message"`
	WorkerID string `json:"worker_id"` // UUID в виде строки
}

// UpdateTaskRequest - запрос на обновление задачи (HTTP)
type UpdateTaskRequest struct {
	Message *string `json:"message,omitempty"`
	Status  *string `json:"status,omitempty"`
	Reason  *string `json:"reason,omitempty"`
}

// TaskResponse - ответ с задачей (HTTP)
type TaskResponse struct {
	ID        string  `json:"id"`
	Message   string  `json:"message"`
	Status    string  `json:"status"`
	WorkerID  string  `json:"worker_id"`
	CreatedBy string  `json:"created_by"`
	Reason    *string `json:"reason,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}
