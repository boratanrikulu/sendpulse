package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/boratanrikulu/sendpulse/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func setupTestDB(t *testing.T) *bun.DB {
	// Using SQLite in-memory for faster test execution
	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	require.NoError(t, err)

	bunDB := bun.NewDB(sqldb, sqlitedialect.New())

	// Create table structure to match production schema
	_, err = bunDB.NewCreateTable().Model((*db.Message)(nil)).Exec(context.Background())
	require.NoError(t, err)

	return bunDB
}

func TestMessageService_GetSentMessages_Pagination(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		pageSize      int
		expectedPage  int
		expectedSize  int
		expectedError error
	}{
		{
			name:          "valid pagination",
			page:          1,
			pageSize:      10,
			expectedPage:  1,
			expectedSize:  10,
			expectedError: nil,
		},
		{
			name:          "page less than 1 defaults to 1", // Prevents errors from invalid API calls
			page:          0,
			pageSize:      10,
			expectedPage:  1,
			expectedSize:  10,
			expectedError: nil,
		},
		{
			name:          "negative page defaults to 1", // Graceful handling of malicious input
			page:          -5,
			pageSize:      10,
			expectedPage:  1,
			expectedSize:  10,
			expectedError: nil,
		},
		{
			name:          "page size 0 uses default", // Standard REST API behavior when not specified
			page:          1,
			pageSize:      0,
			expectedPage:  1,
			expectedSize:  DefaultPageSize,
			expectedError: nil,
		},
		{
			name:          "negative page size returns error", // Security: prevent malicious queries
			page:          1,
			pageSize:      -1,
			expectedError: ErrInvalidPageSize,
		},
		{
			name:          "page size too large returns error", // Prevent memory exhaustion attacks
			page:          1,
			pageSize:      MaxPageSize + 1,
			expectedError: ErrPageSizeTooLarge,
		},
		{
			name:          "page size at max limit is valid", // Boundary testing
			page:          1,
			pageSize:      MaxPageSize,
			expectedPage:  1,
			expectedSize:  MaxPageSize,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB := setupTestDB(t)
			defer testDB.Close()

			service := NewMessageService(testDB)

			result, err := service.GetSentMessages(context.Background(), tt.page, tt.pageSize)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedPage, result.Page)
				assert.Equal(t, tt.expectedSize, result.PageSize)
				assert.Equal(t, "ok", result.Status)
			}
		})
	}
}

func TestMessageService_GetSentMessages_WithData(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Test data includes different statuses to verify filtering
	messages := []*db.Message{
		{
			To:      "+905551111111",
			Content: "Test message 1",
			Status:  db.MessageStatusSent,
			SentAt:  &time.Time{},
		},
		{
			To:      "+905552222222",
			Content: "Test message 2",
			Status:  db.MessageStatusSent,
			SentAt:  &time.Time{},
		},
		{
			To:      "+905553333333",
			Content: "Test message 3",
			Status:  db.MessageStatusPending, // Should be excluded from sent messages
		},
	}

	for _, msg := range messages {
		_, err := testDB.NewInsert().Model(msg).Exec(context.Background())
		require.NoError(t, err)
	}

	service := NewMessageService(testDB)

	result, err := service.GetSentMessages(context.Background(), 1, 20)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Messages)) // Only sent messages should be returned
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PageSize)

	// Verify results are filtered to only sent messages
	// We inserted 3 messages but only 2 were marked as sent
	assert.Equal(t, 2, len(result.Messages))
	for _, msg := range result.Messages {
		assert.Equal(t, "sent", msg.Status)
	}
}

func TestMessageService_GetMessageByID(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Setup test data
	msg := &db.Message{
		To:      "+905551111111",
		Content: "Test message",
		Status:  db.MessageStatusSent,
		SentAt:  &time.Time{},
	}
	_, err := testDB.NewInsert().Model(msg).Exec(context.Background())
	require.NoError(t, err)

	service := NewMessageService(testDB)

	t.Run("valid message ID", func(t *testing.T) {
		result, err := service.GetMessageByID(context.Background(), "1")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "ok", result.Status)
		assert.Equal(t, int64(1), result.Message.ID)
		assert.Equal(t, "+905551111111", result.Message.To)
		assert.Equal(t, "Test message", result.Message.Content)
	})

	t.Run("invalid message ID format", func(t *testing.T) {
		// Testing malformed input handling
		result, err := service.GetMessageByID(context.Background(), "invalid")

		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidMessageID))
		assert.Nil(t, result)
	})

	t.Run("non-existent message ID", func(t *testing.T) {
		// Testing 404 scenario
		result, err := service.GetMessageByID(context.Background(), "999")

		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMessageNotFound))
		assert.Nil(t, result)
	})
}

func TestMessageService_ConvertToMessageResponse(t *testing.T) {
	service := NewMessageService(nil) // No DB needed for pure function

	now := time.Now().UTC()
	webhookResponse := `{"success": true, "message_id": "webhook_123"}`

	msg := &db.Message{
		ID:              123,
		To:              "+905551111111",
		Content:         "Test message",
		Status:          db.MessageStatusSent,
		SentAt:          &now,
		MessageID:       stringPtr("webhook_123"),
		WebhookResponse: &webhookResponse,
		CreatedAt:       now,
	}

	result := service.convertToMessageResponse(msg)

	assert.Equal(t, int64(123), result.ID)
	assert.Equal(t, "+905551111111", result.To)
	assert.Equal(t, "Test message", result.Content)
	assert.Equal(t, "sent", result.Status)
	assert.Equal(t, &now, result.SentAt)
	assert.Equal(t, stringPtr("webhook_123"), result.MessageID)
	assert.Equal(t, now, result.CreatedAt)
	assert.NotNil(t, result.WebhookResponse)

	// Verify JSON parsing works correctly
	assert.NotNil(t, result.WebhookResponse)
	webhookResp := result.WebhookResponse
	assert.Equal(t, true, webhookResp["success"])
	assert.Equal(t, "webhook_123", webhookResp["message_id"])
}

func TestMessageService_ConvertToMessageResponse_InvalidJSON(t *testing.T) {
	service := NewMessageService(nil)

	// Testing resilience to malformed webhook responses in database
	invalidJSON := `{"invalid": json}`
	msg := &db.Message{
		ID:              123,
		To:              "+905551111111",
		Content:         "Test message",
		Status:          db.MessageStatusSent,
		WebhookResponse: &invalidJSON,
	}

	result := service.convertToMessageResponse(msg)

	// Should gracefully handle corruption without crashing
	assert.Nil(t, result.WebhookResponse)
}

func stringPtr(s string) *string {
	return &s
}
