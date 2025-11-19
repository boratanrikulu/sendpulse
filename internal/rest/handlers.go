package rest

import (
	"errors"
	"strconv"
	"time"

	"github.com/boratanrikulu/sendpulse/internal/config"
	"github.com/boratanrikulu/sendpulse/internal/dto"
	"github.com/boratanrikulu/sendpulse/internal/service"
	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	messageService service.MessageInterface
	scheduler      service.SchedulerInterface
}

func NewHandlers(messageService service.MessageInterface, scheduler service.SchedulerInterface) *Handlers {
	return &Handlers{
		messageService: messageService,
		scheduler:      scheduler,
	}
}

// healthHandler handles health check requests
// @Summary Health Check
// @Description Check if the service is running
// @Tags health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Router /api/v1/health [get]
func (h *Handlers) healthHandler(c *fiber.Ctx) error {
	response := &dto.HealthResponse{
		BaseResponse: dto.BaseResponse{
			Status:    "ok",
			Timestamp: time.Now().UTC(),
		},
		Service: "sendpulse",
		Version: config.Version,
		Mode:    string(getCfg(c).Server.Mode),
	}

	return c.JSON(response)
}

// startMessagingHandler handles starting the messaging service
// @Summary Start Messaging Service
// @Description Start the automatic message sending process
// @Tags messaging
// @Produce json
// @Success 200 {object} dto.MessagingControlResponse
// @Failure 400 {object} dto.MessagingControlResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/messaging/start [post]
func (h *Handlers) startMessagingHandler(c *fiber.Ctx) error {
	response, err := h.scheduler.Start(c.Context())
	if err != nil {
		return handleError(c, err)
	}

	statusCode := 200
	if response.Status == "error" {
		statusCode = 400
	}

	return c.Status(statusCode).JSON(response)
}

// stopMessagingHandler handles stopping the messaging service
// @Summary Stop Messaging Service
// @Description Stop the automatic message sending process
// @Tags messaging
// @Produce json
// @Success 200 {object} dto.MessagingControlResponse
// @Failure 400 {object} dto.MessagingControlResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/messaging/stop [post]
func (h *Handlers) stopMessagingHandler(c *fiber.Ctx) error {
	response, err := h.scheduler.Stop(c.Context())
	if err != nil {
		return handleError(c, err)
	}

	statusCode := 200
	if response.Status == "error" {
		statusCode = 400
	}

	return c.Status(statusCode).JSON(response)
}

// messagingStatusHandler handles getting messaging service status
// @Summary Get Messaging Service Status
// @Description Get the current status of the automatic message sending service
// @Tags messaging
// @Produce json
// @Success 200 {object} dto.MessagingStatusResponse
// @Router /api/v1/messaging/status [get]
func (h *Handlers) messagingStatusHandler(c *fiber.Ctx) error {
	response := h.scheduler.GetStatus()
	return c.JSON(response)
}

// listMessagesHandler handles listing sent messages with pagination
// @Summary List Sent Messages
// @Description Get a paginated list of sent messages
// @Tags messages
// @Produce json
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param page_size query int false "Page size (default: 20, max: 100)" minimum(1) maximum(100)
// @Success 200 {object} dto.MessagesListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/messages [get]
func (h *Handlers) listMessagesHandler(c *fiber.Ctx) error {
	// Parse query parameters - let service handle validation
	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil {
			page = p
		}
	}

	pageSize := 20
	if pageSizeParam := c.Query("page_size"); pageSizeParam != "" {
		if ps, err := strconv.Atoi(pageSizeParam); err == nil {
			pageSize = ps
		}
	}

	response, err := h.messageService.GetSentMessages(c.Context(), page, pageSize)
	if err != nil {
		// Handle pagination errors with 400 Bad Request
		if errors.Is(err, service.ErrInvalidPageSize) ||
			errors.Is(err, service.ErrPageSizeTooLarge) ||
			errors.Is(err, service.ErrPageSizeTooSmall) {
			return c.Status(400).JSON(&dto.ErrorResponse{
				BaseResponse: dto.BaseResponse{
					Status:    "error",
					Timestamp: time.Now().UTC(),
				},
				Message: err.Error(),
			})
		}
		return handleError(c, err)
	}

	response.Timestamp = time.Now().UTC()
	return c.JSON(response)
}

// getMessageHandler handles getting a specific message by ID
// @Summary Get Message by ID
// @Description Get details of a specific message by its ID
// @Tags messages
// @Produce json
// @Param id path string true "Message ID"
// @Success 200 {object} dto.SingleMessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/messages/{id} [get]
func (h *Handlers) getMessageHandler(c *fiber.Ctx) error {
	messageID := c.Params("id")
	if messageID == "" {
		return c.Status(400).JSON(&dto.ErrorResponse{
			BaseResponse: dto.BaseResponse{
				Status:    "error",
				Timestamp: time.Now().UTC(),
			},
			Message: "Message ID is required",
		})
	}

	response, err := h.messageService.GetMessageByID(c.Context(), messageID)
	if err != nil {
		if errors.Is(err, service.ErrMessageNotFound) {
			return c.Status(404).JSON(&dto.ErrorResponse{
				BaseResponse: dto.BaseResponse{
					Status:    "error",
					Timestamp: time.Now().UTC(),
				},
				Message: "Message not found",
			})
		}
		if errors.Is(err, service.ErrInvalidMessageID) {
			return c.Status(400).JSON(&dto.ErrorResponse{
				BaseResponse: dto.BaseResponse{
					Status:    "error",
					Timestamp: time.Now().UTC(),
				},
				Message: "Invalid message ID format",
			})
		}
		return handleError(c, err)
	}

	response.Timestamp = time.Now().UTC()
	return c.JSON(response)
}

// Helper functions

func getCfg(c *fiber.Ctx) *config.Cfg {
	return c.Locals("cfg").(*config.Cfg)
}

func handleError(c *fiber.Ctx, err error) error {
	config.Log().Errorf("Handler error: %v", err)

	return c.Status(500).JSON(&dto.ErrorResponse{
		BaseResponse: dto.BaseResponse{
			Status:    "error",
			Timestamp: time.Now().UTC(),
		},
		Message: "Internal server error",
		Error:   err.Error(),
	})
}
