package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/boratanrikulu/sendpulse/internal/db"

	"github.com/uptrace/bun"
)

var (
	sampleMessages = []string{
		"Welcome to our service!",
		"Your order has been confirmed",
		"Don't miss our special offer",
		"Thank you for your purchase",
		"Your payment was successful",
		"Reminder: Your appointment is tomorrow",
		"New features are now available",
		"Your subscription expires soon",
		"Flash sale: 50% off everything",
		"Security alert: Login detected",
		"Your delivery is on the way",
		"Happy birthday! Here's a gift",
		"Limited time offer ends today",
		"Your account has been updated",
		"New message from support team",
	}

	turkishPhoneNumbers = []string{
		"+905551234567", "+905552345678", "+905553456789",
		"+905554567890", "+905555678901", "+905556789012",
		"+905557890123", "+905558901234", "+905559012345",
		"+905550123456", "+905551111111", "+905552222222",
		"+905553333333", "+905554444444", "+905555555555",
	}
)

func seedMessages(ctx context.Context, dbc bun.IDB, count int) error {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Printf("Generating %d random messages...\n", count)

	for i := 0; i < count; i++ {
		message := &db.Message{
			To:      turkishPhoneNumbers[rng.Intn(len(turkishPhoneNumbers))],
			Content: sampleMessages[rng.Intn(len(sampleMessages))],
		}

		if err := db.CreateMessage(ctx, dbc, message); err != nil {
			return fmt.Errorf("failed to create message %d: %w", i+1, err)
		}

		if (i+1)%10 == 0 {
			fmt.Printf("Generated %d messages...\n", i+1)
		}
	}

	fmt.Printf("Successfully generated %d random messages!\n", count)
	return nil
}
