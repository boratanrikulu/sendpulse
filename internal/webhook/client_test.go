package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/boratanrikulu/sendpulse/internal/config"
	"github.com/stretchr/testify/assert"
)

func setupTestClient(serverURL string) *Client {
	cfg := &config.Cfg{
		Webhook: config.Webhook{
			URL: serverURL,
		},
	}
	return NewClient(cfg)
}

func TestClient_SendMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Accepted", "messageId": "test-123"}`))
	}))
	defer server.Close()

	client := setupTestClient(server.URL)
	payload := MessagePayload{
		To:      "+905551111111",
		Content: "Test message",
	}

	response, err := client.SendMessage(context.Background(), payload)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "Accepted", response.Message)
	assert.Equal(t, "test-123", response.MessageID)
}

func TestClient_SendMessage_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Server error"}`))
	}))
	defer server.Close()

	client := setupTestClient(server.URL)
	payload := MessagePayload{
		To:      "+905551111111",
		Content: "Test message",
	}

	response, err := client.SendMessage(context.Background(), payload)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook returned status: 500")
	assert.NotNil(t, response)
	assert.Equal(t, 500, response.StatusCode)
}

func TestClient_SendMessage_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := setupTestClient(server.URL)
	payload := MessagePayload{
		To:      "+905551111111",
		Content: "Test message",
	}

	response, err := client.SendMessage(context.Background(), payload)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "failed to decode response", response.Message)
	assert.Empty(t, response.MessageID)
}

func TestClient_SendMessageWithRetry_Success(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Accepted", "messageId": "retry-123"}`))
	}))
	defer server.Close()

	cfg := &config.Cfg{
		Webhook: config.Webhook{
			URL: server.URL,
		},
		Messaging: config.Messaging{
			MaxRetries: 3,
			RetryDelay: 10 * time.Millisecond,
		},
	}
	client := NewClient(cfg)

	payload := MessagePayload{
		To:      "+905551111111",
		Content: "Test message",
	}

	response, err := client.SendMessageWithRetry(context.Background(), payload)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "Accepted", response.Message)
	assert.Equal(t, "retry-123", response.MessageID)
	assert.Equal(t, 3, attempts)
}

func TestClient_SendMessageWithRetry_MaxRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Cfg{
		Webhook: config.Webhook{
			URL: server.URL,
		},
		Messaging: config.Messaging{
			MaxRetries: 2,
			RetryDelay: 10 * time.Millisecond,
		},
	}
	client := NewClient(cfg)

	payload := MessagePayload{
		To:      "+905551111111",
		Content: "Test message",
	}

	response, err := client.SendMessageWithRetry(context.Background(), payload)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook returned status: 500")
	assert.NotNil(t, response)
	assert.Equal(t, 3, attempts) // 1 initial + 2 retries
}

func TestClient_SendMessageWithRetry_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Cfg{
		Webhook: config.Webhook{
			URL: server.URL,
		},
		Messaging: config.Messaging{
			MaxRetries: 5,
			RetryDelay: 50 * time.Millisecond,
		},
	}
	client := NewClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	payload := MessagePayload{
		To:      "+905551111111",
		Content: "Test message",
	}

	_, err := client.SendMessageWithRetry(ctx, payload)

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}
