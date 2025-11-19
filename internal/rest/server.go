package rest

import (
	"context"
	"fmt"
	"time"

	"github.com/boratanrikulu/sendpulse/internal/config"
	"github.com/boratanrikulu/sendpulse/internal/service"

	"github.com/arsmn/fiber-swagger/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// Server is public rest api service of sendpulse
type Server struct {
	Cfg      *config.Cfg
	handlers *Handlers
	app      *fiber.App
}

// NewServer creates a new Server.
func NewServer(cfg *config.Cfg, messageService *service.MessageService, scheduler *service.Scheduler) *Server {
	return &Server{
		Cfg:      cfg,
		handlers: NewHandlers(messageService, scheduler),
	}
}

// Start runs the rest service.
func (s *Server) Start(ctx context.Context) error {
	s.app = fiber.New(fiber.Config{
		AppName: fmt.Sprintf("%s (mode: %s)", s.Cfg.AppName, s.Cfg.Server.Mode),
	})
	s.app.Use(logger.New(
		logger.Config{
			TimeZone:   time.UTC.String(),
			TimeFormat: time.RFC3339,
		},
	))
	s.app.Use("/", func(c *fiber.Ctx) error {
		c.Locals("cfg", s.Cfg)
		return c.Next()
	})
	s.applyRouting()

	config.Log().Infof("Starting SendPulse server on %s", s.Cfg.Server.Address)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		config.Log().Info("Shutting down SendPulse server...")
		if err := s.app.Shutdown(); err != nil {
			config.Log().Errorf("Server shutdown error: %v", err)
		}
	}()

	config.Log().Info("SendPulse server started successfully")
	return s.app.Listen(s.Cfg.Server.Address)
}

func (s *Server) applyRouting() {
	// Swagger documentation endpoint
	s.app.Get("/swagger/*", swagger.HandlerDefault)

	api := s.app.Group("/api/v1")

	api.Get("/health", s.handlers.healthHandler)

	// Messaging control endpoints
	api.Post("/messaging/start", s.handlers.startMessagingHandler)
	api.Post("/messaging/stop", s.handlers.stopMessagingHandler)
	api.Get("/messaging/status", s.handlers.messagingStatusHandler)

	// Message history endpoints
	api.Get("/messages", s.handlers.listMessagesHandler)
	api.Get("/messages/:id", s.handlers.getMessageHandler)
}
