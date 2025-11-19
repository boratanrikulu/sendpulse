package rest

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/boratanrikulu/sendpulse/internal/config"
	"github.com/boratanrikulu/sendpulse/internal/dto"
	"github.com/boratanrikulu/sendpulse/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMessage implements service interface for testing
type MockMessage struct {
	mock.Mock
}

func (m *MockMessage) GetSentMessages(ctx context.Context, page, pageSize int) (*dto.MessagesListResponse, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.MessagesListResponse), args.Error(1)
}

func (m *MockMessage) GetMessageByID(ctx context.Context, id string) (*dto.SingleMessageResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.SingleMessageResponse), args.Error(1)
}

type MockScheduler struct {
	mock.Mock
}

func (m *MockScheduler) Start(ctx context.Context) (*dto.MessagingControlResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*dto.MessagingControlResponse), args.Error(1)
}

func (m *MockScheduler) Stop(ctx context.Context) (*dto.MessagingControlResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*dto.MessagingControlResponse), args.Error(1)
}

func (m *MockScheduler) GetStatus() *dto.MessagingStatusResponse {
	args := m.Called()
	return args.Get(0).(*dto.MessagingStatusResponse)
}

func (m *MockScheduler) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

func setupTestApp() (*fiber.App, *MockMessage, *MockScheduler) {
	cfg := &config.Cfg{
		AppName: "sendpulse",
		Server: config.Server{
			Mode: config.ModeDev,
		},
	}

	mockMessage := &MockMessage{}
	mockScheduler := &MockScheduler{}

	handlers := NewHandlers(mockMessage, mockScheduler)

	app := fiber.New()
	// Simulate middleware that sets config in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("cfg", cfg)
		return c.Next()
	})

	api := app.Group("/api/v1")
	api.Get("/health", handlers.healthHandler)
	api.Post("/messaging/start", handlers.startMessagingHandler)
	api.Post("/messaging/stop", handlers.stopMessagingHandler)
	api.Get("/messaging/status", handlers.messagingStatusHandler)
	api.Get("/messages", handlers.listMessagesHandler)
	api.Get("/messages/:id", handlers.getMessageHandler)

	return app, mockMessage, mockScheduler
}

func TestHandlers_Health(t *testing.T) {
	app, _, _ := setupTestApp()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Health endpoint should always work regardless of service state
}

func TestHandlers_ListMessages(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		expectedResponse := &dto.MessagesListResponse{
			BaseResponse: dto.BaseResponse{Status: "ok"},
			Messages:     []dto.MessageResponse{},
			Total:        0,
			Page:         1,
			PageSize:     20,
		}

		mockMessage.On("GetSentMessages", mock.Anything, 1, 20).Return(expectedResponse, nil)

		req := httptest.NewRequest("GET", "/api/v1/messages", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("custom pagination parameters", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		expectedResponse := &dto.MessagesListResponse{
			BaseResponse: dto.BaseResponse{Status: "ok"},
			Messages:     []dto.MessageResponse{},
			Total:        0,
			Page:         2,
			PageSize:     10,
		}

		// Should parse query parameters correctly
		mockMessage.On("GetSentMessages", mock.Anything, 2, 10).Return(expectedResponse, nil)

		req := httptest.NewRequest("GET", "/api/v1/messages?page=2&page_size=10", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("invalid page size error", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		// Testing pagination validation error handling
		mockMessage.On("GetSentMessages", mock.Anything, 1, -1).Return(nil, service.ErrInvalidPageSize)

		req := httptest.NewRequest("GET", "/api/v1/messages?page_size=-1", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode) // Should return 400 for validation errors
		mockMessage.AssertExpectations(t)
	})

	t.Run("page size too large error", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		mockMessage.On("GetSentMessages", mock.Anything, 1, 1000).Return(nil, service.ErrPageSizeTooLarge)

		req := httptest.NewRequest("GET", "/api/v1/messages?page_size=1000", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestHandlers_GetMessage(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		expectedResponse := &dto.SingleMessageResponse{
			BaseResponse: dto.BaseResponse{Status: "ok"},
			Message: dto.MessageResponse{
				ID:      123,
				To:      "+905551111111",
				Content: "Test message",
				Status:  "sent",
			},
		}

		mockMessage.On("GetMessageByID", mock.Anything, "123").Return(expectedResponse, nil)

		req := httptest.NewRequest("GET", "/api/v1/messages/123", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("message not found", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		// Testing 404 error handling
		mockMessage.On("GetMessageByID", mock.Anything, "999").Return(nil, service.ErrMessageNotFound)

		req := httptest.NewRequest("GET", "/api/v1/messages/999", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("invalid message ID", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		// Testing malformed ID handling
		mockMessage.On("GetMessageByID", mock.Anything, "invalid").Return(nil, service.ErrInvalidMessageID)

		req := httptest.NewRequest("GET", "/api/v1/messages/invalid", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("empty message ID", func(t *testing.T) {
		app, _, _ := setupTestApp()
		// Should handle missing ID parameter - test with malformed URL that won't match any route
		req := httptest.NewRequest("GET", "/api/v1/messages//invalid", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		// Should return 404 for unmatched route
		assert.Equal(t, 404, resp.StatusCode)
	})
}

func TestHandlers_MessagingControl(t *testing.T) {
	t.Run("start messaging success", func(t *testing.T) {
		app, _, mockScheduler := setupTestApp()
		expectedResponse := &dto.MessagingControlResponse{
			BaseResponse: dto.BaseResponse{
				Status:    "success",
				Timestamp: time.Now().UTC(),
			},
			Message: "Messaging service started successfully",
		}

		mockScheduler.On("Start", mock.Anything).Return(expectedResponse, nil)

		req := httptest.NewRequest("POST", "/api/v1/messaging/start", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		mockScheduler.AssertExpectations(t)
	})

	t.Run("start messaging already running", func(t *testing.T) {
		app, _, mockScheduler := setupTestApp()
		// Service should handle duplicate start gracefully
		expectedResponse := &dto.MessagingControlResponse{
			BaseResponse: dto.BaseResponse{
				Status:    "error",
				Timestamp: time.Now().UTC(),
			},
			Message: "Messaging service is already running",
		}

		mockScheduler.On("Start", mock.Anything).Return(expectedResponse, nil)

		req := httptest.NewRequest("POST", "/api/v1/messaging/start", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode) // Should return 400 for error status
		mockScheduler.AssertExpectations(t)
	})

	t.Run("stop messaging success", func(t *testing.T) {
		app, _, mockScheduler := setupTestApp()
		expectedResponse := &dto.MessagingControlResponse{
			BaseResponse: dto.BaseResponse{
				Status:    "success",
				Timestamp: time.Now().UTC(),
			},
			Message: "Messaging service stopped successfully",
		}

		mockScheduler.On("Stop", mock.Anything).Return(expectedResponse, nil)

		req := httptest.NewRequest("POST", "/api/v1/messaging/stop", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		mockScheduler.AssertExpectations(t)
	})

	t.Run("messaging status", func(t *testing.T) {
		app, _, mockScheduler := setupTestApp()
		expectedResponse := &dto.MessagingStatusResponse{
			BaseResponse: dto.BaseResponse{
				Status:    "ok",
				Timestamp: time.Now().UTC(),
			},
			Enabled:    true,
			Interval:   "2m0s",
			BatchSize:  2,
			MaxRetries: 3,
			RetryDelay: "30s",
		}

		mockScheduler.On("GetStatus").Return(expectedResponse)

		req := httptest.NewRequest("GET", "/api/v1/messaging/status", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		mockScheduler.AssertExpectations(t)
	})
}

func TestHandlers_ErrorHandling(t *testing.T) {
	app, mockMessage, _ := setupTestApp()

	t.Run("database connection error", func(t *testing.T) {
		// Testing infrastructure failure handling
		dbError := errors.New("database connection failed")
		mockMessage.On("GetSentMessages", mock.Anything, 1, 20).Return(nil, dbError)

		req := httptest.NewRequest("GET", "/api/v1/messages", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode) // Should return 500 for unexpected errors
		mockMessage.AssertExpectations(t)
	})
}

func TestHandlers_QueryParameterParsing(t *testing.T) {
	t.Run("valid parameters parsed correctly", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		expectedResponse := &dto.MessagesListResponse{
			BaseResponse: dto.BaseResponse{Status: "ok"},
			Messages:     []dto.MessageResponse{},
			Total:        0,
			Page:         2,
			PageSize:     50,
		}

		// Handler should pass parsed values to service
		mockMessage.On("GetSentMessages", mock.Anything, 2, 50).Return(expectedResponse, nil)

		req := httptest.NewRequest("GET", "/api/v1/messages?page=2&page_size=50", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("invalid parameters use defaults", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		expectedResponse := &dto.MessagesListResponse{
			BaseResponse: dto.BaseResponse{Status: "ok"},
			Messages:     []dto.MessageResponse{},
			Total:        0,
			Page:         1,
			PageSize:     20,
		}

		// Handler uses defaults for unparseable values
		mockMessage.On("GetSentMessages", mock.Anything, 1, 20).Return(expectedResponse, nil)

		req := httptest.NewRequest("GET", "/api/v1/messages?page=invalid&page_size=invalid", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("service normalizes zero values", func(t *testing.T) {
		app, mockMessage, _ := setupTestApp()
		expectedResponse := &dto.MessagesListResponse{
			BaseResponse: dto.BaseResponse{Status: "ok"},
			Messages:     []dto.MessageResponse{},
			Total:        0,
			Page:         1,  // Service normalized 0 to 1
			PageSize:     20, // Service normalized 0 to default
		}

		// Handler passes 0 values, service normalizes them
		mockMessage.On("GetSentMessages", mock.Anything, 0, 0).Return(expectedResponse, nil)

		req := httptest.NewRequest("GET", "/api/v1/messages?page=0&page_size=0", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}
