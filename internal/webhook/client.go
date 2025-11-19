package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/boratanrikulu/sendpulse/internal/config"
)

type MessagePayload struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

type Response struct {
	StatusCode int       `json:"status_code"`
	Message    string    `json:"message"`
	MessageID  string    `json:"message_id"`
	Timestamp  time.Time `json:"timestamp"`
}

type Client struct {
	httpClient *http.Client
	cfg        *config.Cfg
}

func NewClient(cfg *config.Cfg) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cfg: cfg,
	}
}

func (c *Client) SendMessage(ctx context.Context, payload MessagePayload) (*Response, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.Webhook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	var responseBody struct {
		Message   string `json:"message"`
		MessageID string `json:"messageId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		responseBody.Message = "failed to decode response"
	}

	webhookResponse := &Response{
		StatusCode: resp.StatusCode,
		Message:    responseBody.Message,
		MessageID:  responseBody.MessageID,
		Timestamp:  time.Now().UTC(),
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return webhookResponse, fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	return webhookResponse, nil
}

func (c *Client) SendMessageWithRetry(ctx context.Context, payload MessagePayload) (*Response, error) {
	var lastErr error
	var lastResponse *Response

	maxRetries := c.cfg.Messaging.MaxRetries
	retryDelay := c.cfg.Messaging.RetryDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return lastResponse, ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		response, err := c.SendMessage(ctx, payload)
		if err == nil {
			return response, nil
		}

		lastErr = err
		lastResponse = response
	}

	return lastResponse, lastErr
}
