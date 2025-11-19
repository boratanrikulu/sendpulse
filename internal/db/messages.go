package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

type MessageStatus string

const (
	MessageStatusPending MessageStatus = "pending"
	MessageStatusSending MessageStatus = "sending"
	MessageStatusSent    MessageStatus = "sent"
	MessageStatusFailed  MessageStatus = "failed"
	MaxMessageLength     int           = 160
)

var (
	ErrMessageTooLong = errors.New("message content exceeds maximum length")
)

type Message struct {
	bun.BaseModel `bun:"table:messages"`

	ID              int64         `bun:"id,pk,autoincrement" json:"id"`
	To              string        `bun:"to,notnull" json:"to"`
	Content         string        `bun:"content,notnull" json:"content"`
	Status          MessageStatus `bun:"status,notnull,default:'pending'" json:"status"`
	SentAt          *time.Time    `bun:"sent_at,nullzero" json:"sent_at,omitempty"`
	MessageID       *string       `bun:"message_id,nullzero" json:"message_id,omitempty"`
	WebhookResponse *string       `bun:"webhook_response,type:jsonb,nullzero" json:"webhook_response,omitempty"`
	CreatedAt       time.Time     `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time     `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

// CreateMessage inserts a new message into the database
func CreateMessage(ctx context.Context, db bun.IDB, message *Message) error {
	if len(message.Content) > MaxMessageLength {
		return ErrMessageTooLong
	}

	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()
	message.Status = MessageStatusPending

	_, err := db.NewInsert().Model(message).Exec(ctx)
	return err
}

// ClaimNextMessage atomically claims the next available message for processing
func ClaimNextMessage(ctx context.Context, db bun.IDB) (*Message, error) {
	message := new(Message)
	now := time.Now()

	query := `
		UPDATE messages 
		SET status = ?, 
		    updated_at = ?
		WHERE id = (
			SELECT id FROM messages 
			WHERE status = ?
			ORDER BY created_at ASC 
			FOR UPDATE SKIP LOCKED 
			LIMIT 1
		) 
		RETURNING *`

	err := db.NewRaw(query,
		MessageStatusSending,
		now,
		MessageStatusPending).Scan(ctx, message)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if message.ID == 0 {
		return nil, nil
	}

	return message, nil
}

// UpdateMessageStatus updates the status of a message and optionally sets sent_at and message_id
func UpdateMessageStatus(ctx context.Context, db bun.IDB, messageID int64, status MessageStatus, sentAt *time.Time, webhookMessageID *string, webhookResponse *string) error {
	query := db.NewUpdate().
		Model(&Message{}).
		Set("status = ?", status).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", messageID)

	if sentAt != nil {
		query = query.Set("sent_at = ?", *sentAt)
	}

	if webhookMessageID != nil {
		query = query.Set("message_id = ?", *webhookMessageID)
	}

	if webhookResponse != nil {
		query = query.Set("webhook_response = ?", *webhookResponse)
	}

	_, err := query.Exec(ctx)
	return err
}

// GetSentMessages retrieves all sent messages with pagination
func GetSentMessages(ctx context.Context, db bun.IDB, limit, offset int) ([]*Message, error) {
	var messages []*Message

	err := db.NewSelect().
		Model(&messages).
		Where("status = ?", MessageStatusSent).
		Order("sent_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	return messages, err
}

// GetMessageByID retrieves a single message by its ID
func GetMessageByID(ctx context.Context, db bun.IDB, id int64) (*Message, error) {
	message := &Message{}

	err := db.NewSelect().
		Model(message).
		Where("id = ?", id).
		Scan(ctx)

	return message, err
}

// GetTotalSentMessagesCount returns the total count of sent messages
func GetTotalSentMessagesCount(ctx context.Context, db bun.IDB) (int, error) {
	count, err := db.NewSelect().
		Model(&Message{}).
		Where("status = ?", MessageStatusSent).
		Count(ctx)

	return count, err
}
