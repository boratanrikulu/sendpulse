package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/boratanrikulu/sendpulse/internal/db"
	"github.com/boratanrikulu/sendpulse/internal/dto"
	"github.com/uptrace/bun"
)

// Pagination constants
const (
	// DefaultPageSize is the default number of messages returned per page when not specified
	DefaultPageSize = 20
	// MaxPageSize is the maximum number of messages that can be returned per page
	// This prevents excessive memory usage and ensures reasonable response times
	MaxPageSize = 100
	// MinPageSize is the minimum page size allowed (must be at least 1)
	MinPageSize = 1
	// MinPage is the minimum page number (pages start from 1)
	MinPage = 1
)

// Pagination errors
var (
	ErrInvalidPageSize  = errors.New("page size cannot be negative")
	ErrPageSizeTooLarge = fmt.Errorf("page size cannot exceed %d", MaxPageSize)
	ErrPageSizeTooSmall = fmt.Errorf("page size must be at least %d", MinPageSize)
	ErrMessageNotFound  = errors.New("message not found")
	ErrInvalidMessageID = errors.New("invalid message ID format")
)

// MessageInterface defines message-related operations
type MessageInterface interface {
	GetSentMessages(ctx context.Context, page, pageSize int) (*dto.MessagesListResponse, error)
	GetMessageByID(ctx context.Context, id string) (*dto.SingleMessageResponse, error)
}

type MessageService struct {
	db *bun.DB
}

func NewMessageService(database *bun.DB) *MessageService {
	return &MessageService{
		db: database,
	}
}

// GetSentMessages retrieves paginated sent messages
// Parameters:
// - page: Page number (starts from 1, defaults to 1 if < 1)
// - pageSize: Number of messages per page (0 = default, must be between 1-100)
// Returns error if pageSize is invalid (negative or > 100)
func (s *MessageService) GetSentMessages(ctx context.Context, page, pageSize int) (*dto.MessagesListResponse, error) {
	// Validate and normalize page number
	// Pages start from 1, so anything less than 1 defaults to first page
	if page < MinPage {
		page = MinPage
	}

	// Validate and normalize page size
	if pageSize < 0 {
		return nil, ErrInvalidPageSize
	}
	if pageSize == 0 {
		// If pageSize is 0, use the default page size
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		return nil, ErrPageSizeTooLarge
	}
	if pageSize < MinPageSize {
		return nil, ErrPageSizeTooSmall
	}

	offset := (page - 1) * pageSize

	// Get messages
	messages, err := db.GetSentMessages(ctx, s.db, pageSize, offset)
	if err != nil {
		return nil, err
	}

	// Get total count
	total, err := db.GetTotalSentMessagesCount(ctx, s.db)
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	messageResponses := make([]dto.MessageResponse, len(messages))
	for i, msg := range messages {
		messageResponses[i] = s.convertToMessageResponse(msg)
	}

	return &dto.MessagesListResponse{
		BaseResponse: dto.BaseResponse{
			Status: "ok",
		},
		Messages: messageResponses,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetMessageByID retrieves a single message by its ID
func (s *MessageService) GetMessageByID(ctx context.Context, id string) (*dto.SingleMessageResponse, error) {
	messageID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMessageID, err.Error())
	}

	message, err := db.GetMessageByID(ctx, s.db, messageID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrMessageNotFound, err.Error())
	}

	return &dto.SingleMessageResponse{
		BaseResponse: dto.BaseResponse{
			Status: "ok",
		},
		Message: s.convertToMessageResponse(message),
	}, nil
}

// convertToMessageResponse converts db.Message to dto.MessageResponse
func (s *MessageService) convertToMessageResponse(msg *db.Message) dto.MessageResponse {
	response := dto.MessageResponse{
		ID:        msg.ID,
		To:        msg.To,
		Content:   msg.Content,
		Status:    string(msg.Status),
		SentAt:    msg.SentAt,
		MessageID: msg.MessageID,
		CreatedAt: msg.CreatedAt,
	}

	// Parse webhook response if exists
	if msg.WebhookResponse != nil {
		var webhookResp map[string]any
		if err := json.Unmarshal([]byte(*msg.WebhookResponse), &webhookResp); err == nil {
			response.WebhookResponse = webhookResp
		}
	}

	return response
}
