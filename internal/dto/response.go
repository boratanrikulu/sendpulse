package dto

import "time"

// BaseResponse contains common response fields
type BaseResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	BaseResponse
	Service string `json:"service"`
	Version string `json:"version"`
	Mode    string `json:"mode"`
}

// MessageResponse represents a single message
type MessageResponse struct {
	ID              int64          `json:"id"`
	To              string         `json:"to"`
	Content         string         `json:"content"`
	Status          string         `json:"status"`
	SentAt          *time.Time     `json:"sent_at,omitempty"`
	MessageID       *string        `json:"message_id,omitempty"`
	WebhookResponse map[string]any `json:"webhook_response,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

// MessagesListResponse represents paginated messages list
type MessagesListResponse struct {
	BaseResponse
	Messages []MessageResponse `json:"messages"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// SingleMessageResponse represents single message response
type SingleMessageResponse struct {
	BaseResponse
	Message MessageResponse `json:"message"`
}

// MessagingControlResponse represents messaging control operation response
type MessagingControlResponse struct {
	BaseResponse
	Message string `json:"message"`
}

// MessagingStatusResponse represents messaging service status
type MessagingStatusResponse struct {
	BaseResponse
	Enabled    bool   `json:"enabled"`
	Interval   string `json:"interval"`
	BatchSize  int    `json:"batch_size"`
	MaxRetries int    `json:"max_retries"`
	RetryDelay string `json:"retry_delay"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	BaseResponse
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}
