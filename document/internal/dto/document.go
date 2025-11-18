package dto

type AddDocumentRequest struct {
	TaskID      string   `json:"task_id"`
	Filename    string   `json:"filename"`
	ContentType string   `json:"content_type"`
	OwnerID     string   `json:"owner_id"`
	Tags        []string `json:"tags"`
	Data        string   `json:"data"` // base64 encoded content
}

type DocumentResponse struct {
	ID          string   `json:"id"`
	TaskID      string   `json:"task_id"`
	OwnerID     string   `json:"owner_id"`
	Filename    string   `json:"filename"`
	ContentType string   `json:"content_type"`
	Size        int64    `json:"size"`
	Tags        []string `json:"tags"`
	UploadedAt  int64    `json:"uploaded_at"`
	Checksum    string   `json:"checksum,omitempty"`
}
