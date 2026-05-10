package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"notification-service/internal/broker"
	"notification-service/internal/domain"
	"notification-service/internal/provider"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const (
	maxRetries = 5
	baseDelay  = 2 * time.Second
)

// NotificationConsumer listens to the payment.completed queue
// and delivers emails via the injected EmailSender (Adapter Pattern).
type NotificationConsumer struct {
	ch       *amqp.Channel
	rdb      *redis.Client // Redis for idempotency
	sender   provider.EmailSender
}

func NewNotificationConsumer(ch *amqp.Channel, rdb *redis.Client, sender provider.EmailSender) *NotificationConsumer {
	return &NotificationConsumer{ch: ch, rdb: rdb, sender: sender}
}

// Start begins consuming messages. Blocks until stopCh is closed.
func (c *NotificationConsumer) Start(stopCh <-chan struct{}) error {
	msgs, err := c.ch.Consume(
		broker.QueueName,
		"notification-service",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("[Consumer] Listening on queue: %s", broker.QueueName)

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				log.Println("[Consumer] Delivery channel closed")
				return nil
			}
			c.handleMessage(msg)

		case <-stopCh:
			log.Println("[Consumer] Stop signal received, shutting down")
			return nil
		}
	}
}

func (c *NotificationConsumer) handleMessage(msg amqp.Delivery) {
	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[Consumer] Failed to parse message: %v — rejecting to DLQ", err)
		msg.Nack(false, false)
		return
	}

	if event.EventID == "" {
		log.Printf("[Consumer] Missing event_id — rejecting")
		msg.Nack(false, false)
		return
	}

	// --- Idempotency check via Redis ---
	ctx := context.Background()
	idempKey := fmt.Sprintf("notif:processed:%s", event.EventID)

	exists, err := c.rdb.Exists(ctx, idempKey).Result()
	if err != nil {
		log.Printf("[Consumer] Redis idempotency check error: %v — requeuing", err)
		msg.Nack(false, true)
		return
	}
	if exists > 0 {
		log.Printf("[Consumer] Duplicate event %s — skipping (idempotent)", event.EventID)
		msg.Ack(false)
		return
	}

	// --- Send with exponential backoff retry ---
	if err := c.sendWithRetry(event); err != nil {
		log.Printf("[Consumer] All retries exhausted for event %s: %v — sending to DLQ", event.EventID, err)
		msg.Nack(false, false) // reject → DLQ
		return
	}

	// --- Mark as processed in Redis (24h TTL) ---
	if err := c.rdb.Set(ctx, idempKey, "done", 24*time.Hour).Err(); err != nil {
		log.Printf("[Consumer] Warning: failed to mark event %s as processed in Redis: %v", event.EventID, err)
		// Still ACK — email was sent. A second delivery would be a harmless duplicate log.
	}

	msg.Ack(false)
	log.Printf("[Consumer] Successfully processed event %s", event.EventID)
}

// sendWithRetry calls the provider with exponential backoff.
// Delays: 2s, 4s, 8s, 16s, 32s (for maxRetries = 5).
func (c *NotificationConsumer) sendWithRetry(event domain.PaymentEvent) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
			log.Printf("[Consumer] Retry %d/%d for event %s — waiting %s", attempt, maxRetries, event.EventID, delay)
			time.Sleep(delay)
		}

		if err := c.sender.Send(event); err != nil {
			lastErr = err
			log.Printf("[Consumer] Attempt %d failed for event %s: %v", attempt+1, event.EventID, err)
			continue
		}
		return nil // success
	}
	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}
