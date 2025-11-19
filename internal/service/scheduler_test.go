package service

import (
	"context"
	"testing"
	"time"

	"github.com/boratanrikulu/sendpulse/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestScheduler_StartStop(t *testing.T) {
	cfg := &config.Cfg{
		Messaging: config.Messaging{
			Interval:   2 * time.Minute,
			BatchSize:  2,
			MaxRetries: 3,
			RetryDelay: 30 * time.Second,
		},
	}

	service := NewScheduler(nil, cfg) // No DB needed for control operations

	t.Run("start service when stopped", func(t *testing.T) {
		response, err := service.Start(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Contains(t, response.Message, "started successfully")
		assert.True(t, service.IsRunning())
	})

	t.Run("start service when already running", func(t *testing.T) {
		// Should handle duplicate start gracefully
		response, err := service.Start(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "error", response.Status) // Indicates already running
		assert.Contains(t, response.Message, "already running")
		assert.True(t, service.IsRunning())
	})

	t.Run("stop running service", func(t *testing.T) {
		response, err := service.Stop(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "success", response.Status)
		assert.Contains(t, response.Message, "stopped successfully")
		assert.False(t, service.IsRunning())
	})

	t.Run("stop service when already stopped", func(t *testing.T) {
		// Should handle duplicate stop gracefully
		response, err := service.Stop(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "error", response.Status) // Indicates not running
		assert.Contains(t, response.Message, "not running")
		assert.False(t, service.IsRunning())
	})
}

func TestScheduler_GetStatus(t *testing.T) {
	cfg := &config.Cfg{
		Messaging: config.Messaging{
			Interval:   2 * time.Minute,
			BatchSize:  2,
			MaxRetries: 3,
			RetryDelay: 30 * time.Second,
		},
	}

	service := NewScheduler(nil, cfg)

	t.Run("status when stopped", func(t *testing.T) {
		response := service.GetStatus()

		assert.Equal(t, "ok", response.Status)
		assert.False(t, response.Enabled)
		assert.Equal(t, "2m0s", response.Interval)
		assert.Equal(t, 2, response.BatchSize)
		assert.Equal(t, 3, response.MaxRetries)
		assert.Equal(t, "30s", response.RetryDelay)
	})

	t.Run("status when running", func(t *testing.T) {
		// Start service to change state
		_, err := service.Start(context.Background())
		assert.NoError(t, err)

		response := service.GetStatus()

		assert.Equal(t, "ok", response.Status)
		assert.True(t, response.Enabled)

		// Cleanup
		_, _ = service.Stop(context.Background())
	})
}

func TestScheduler_IsRunning_ThreadSafety(t *testing.T) {
	// Testing concurrent access to running state
	cfg := &config.Cfg{
		Messaging: config.Messaging{
			Interval: 100 * time.Millisecond, // Short interval for testing
		},
	}

	service := NewScheduler(nil, cfg)

	// Start goroutines that check running state concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				// Should never panic due to race conditions
				_ = service.IsRunning()
			}
			done <- true
		}()
	}

	// Change state while readers are active
	_, _ = service.Start(context.Background())
	time.Sleep(10 * time.Millisecond)
	_, _ = service.Stop(context.Background())

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test passes if no race conditions occurred
}

func TestScheduler_ContextCancellation(t *testing.T) {
	cfg := &config.Cfg{
		Messaging: config.Messaging{
			Interval: 50 * time.Millisecond, // Very short for testing
		},
	}

	service := NewScheduler(nil, cfg)

	// Create context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start service with cancellable context
	_, err := service.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Wait for context cancellation to stop processing
	time.Sleep(150 * time.Millisecond)

	// Service should still be marked as running until explicitly stopped
	// (Context cancellation affects processing loop, not service state)
	assert.True(t, service.IsRunning())

	// Cleanup
	_, _ = service.Stop(context.Background())
}
