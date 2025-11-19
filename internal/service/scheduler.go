package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/boratanrikulu/sendpulse/internal/config"
	"github.com/boratanrikulu/sendpulse/internal/db"
	"github.com/boratanrikulu/sendpulse/internal/dto"
	"github.com/boratanrikulu/sendpulse/internal/webhook"
	"github.com/uptrace/bun"
)

const MAXIMUM_MESSAGE_SENDING_TIME = 5 * time.Second

// SchedulerInterface defines messaging scheduler control operations
type SchedulerInterface interface {
	Start(ctx context.Context) (*dto.MessagingControlResponse, error)
	Stop(ctx context.Context) (*dto.MessagingControlResponse, error)
	GetStatus() *dto.MessagingStatusResponse
	IsRunning() bool
}

// Scheduler handles the automatic message sending functionality
type Scheduler struct {
	db            *bun.DB
	cfg           *config.Cfg
	webhookClient *webhook.Client
	running       bool
	stopCh        chan struct{}
	mu            sync.RWMutex
}

func NewScheduler(database *bun.DB, cfg *config.Cfg) *Scheduler {
	return &Scheduler{
		db:            database,
		cfg:           cfg,
		webhookClient: webhook.NewClient(cfg),
		stopCh:        make(chan struct{}),
	}
}

// Start begins the automatic message sending process
func (s *Scheduler) Start(ctx context.Context) (*dto.MessagingControlResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return &dto.MessagingControlResponse{
			BaseResponse: dto.BaseResponse{
				Status:    "error",
				Timestamp: time.Now().UTC(),
			},
			Message: "Messaging service is already running",
		}, nil
	}

	s.running = true
	s.stopCh = make(chan struct{})

	// Start the message processing loop in a goroutine
	go s.processMessages(ctx)

	config.Log().Info("Messaging service started")

	return &dto.MessagingControlResponse{
		BaseResponse: dto.BaseResponse{
			Status:    "success",
			Timestamp: time.Now().UTC(),
		},
		Message: "Messaging service started successfully",
	}, nil
}

// Stop halts the automatic message sending process
func (s *Scheduler) Stop(ctx context.Context) (*dto.MessagingControlResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return &dto.MessagingControlResponse{
			BaseResponse: dto.BaseResponse{
				Status:    "error",
				Timestamp: time.Now().UTC(),
			},
			Message: "Messaging service is not running",
		}, nil
	}

	s.running = false
	close(s.stopCh)

	config.Log().Info("Messaging service stopped")

	return &dto.MessagingControlResponse{
		BaseResponse: dto.BaseResponse{
			Status:    "success",
			Timestamp: time.Now().UTC(),
		},
		Message: "Messaging service stopped successfully",
	}, nil
}

// GetStatus returns the current status of the messaging service
func (s *Scheduler) GetStatus() *dto.MessagingStatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &dto.MessagingStatusResponse{
		BaseResponse: dto.BaseResponse{
			Status:    "ok",
			Timestamp: time.Now().UTC(),
		},
		Enabled:    s.running,
		Interval:   s.cfg.Messaging.Interval.String(),
		BatchSize:  s.cfg.Messaging.BatchSize,
		MaxRetries: s.cfg.Messaging.MaxRetries,
		RetryDelay: s.cfg.Messaging.RetryDelay.String(),
	}
}

// IsRunning returns whether the messaging service is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// processMessages is the main message processing loop
func (s *Scheduler) processMessages(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.Messaging.Interval)
	defer ticker.Stop()

	if !s.cfg.Messaging.Enabled {
		return
	}

	config.Log().Info("Message processing loop started")

	for {
		select {
		case <-ctx.Done():
			config.Log().Info("Message processing stopped due to context cancellation")
			return
		case <-s.stopCh:
			config.Log().Info("Message processing stopped")
			return
		case <-ticker.C:
			s.processBatch(ctx)
		}
	}
}

// processBatch processes a batch of messages
func (s *Scheduler) processBatch(ctx context.Context) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.cfg.Messaging.BatchSize)

	config.Log().Infof("Processing messages")

	var sentCount int
	for i := 0; i < s.cfg.Messaging.BatchSize; i++ {
		message, err := db.ClaimNextMessage(ctx, s.db)
		if err != nil {
			config.Log().Errorf("Failed to claim message: %v", err)
			continue
		}

		if message == nil {
			break
		}

		wg.Add(1)
		sentCount++
		go func(msg *db.Message) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			s.processMessage(ctx, msg)
		}(message)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		config.Log().Info("Batch processing cancelled")
	case <-done:
		config.Log().Infof("Batch processing completed, proceed %d messages", sentCount)
	}
}

func (s *Scheduler) processMessage(ctx context.Context, message *db.Message) {
	payload := webhook.MessagePayload{
		To:      message.To,
		Content: message.Content,
	}

	cctx, cancel := context.WithTimeout(ctx, MAXIMUM_MESSAGE_SENDING_TIME)
	defer cancel()
	response, err := s.webhookClient.SendMessageWithRetry(cctx, payload)
	if err != nil {
		config.Log().Errorf("Failed to send message %d: %v", message.ID, err)
		if updateErr := db.UpdateMessageStatus(ctx, s.db, message.ID, db.MessageStatusFailed, nil, nil, nil); updateErr != nil {
			config.Log().Errorf("Failed to update message %d to failed status: %v", message.ID, updateErr)
		}
		return
	}

	responseJSON, _ := json.Marshal(response)
	responseStr := string(responseJSON)
	messageID := response.MessageID
	now := time.Now().UTC()

	if err := db.UpdateMessageStatus(ctx, s.db, message.ID, db.MessageStatusSent, &now, &messageID, &responseStr); err != nil {
		config.Log().Errorf("Failed to update message %d status: %v", message.ID, err)
	}

	config.Log().Debugf("Message %d sent successfully to %s", message.ID, message.To)
}
