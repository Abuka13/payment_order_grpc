package consumer

import (
	"encoding/json"
	"fmt"
	"log"
	"notification-service/internal/broker"
	"notification-service/internal/domain"
	"notification-service/internal/idempotency"

	amqp "github.com/rabbitmq/amqp091-go"
)

// NotificationConsumer listens to the payment.completed queue and simulates email sending.
type NotificationConsumer struct {
	ch    *amqp.Channel
	store *idempotency.Store
}

func NewNotificationConsumer(ch *amqp.Channel, store *idempotency.Store) *NotificationConsumer {
	return &NotificationConsumer{ch: ch, store: store}
}

// Start begins consuming messages. Blocks until the channel is closed.
// stopCh is used for graceful shutdown signalling.
func (c *NotificationConsumer) Start(stopCh <-chan struct{}) error {
	msgs, err := c.ch.Consume(
		broker.QueueName,
		"notification-service", // consumer tag
		false,                  // auto-ack = false (manual ACK required)
		false,                  // exclusive
		false,                  // no-local
		false,                  // no-wait
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
			log.Println("[Consumer] Stop signal received, shutting down consumer")
			return nil
		}
	}
}

func (c *NotificationConsumer) handleMessage(msg amqp.Delivery) {
	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[Consumer] Failed to parse message: %v — sending to DLQ", err)
		// Non-recoverable: reject without requeue → goes to DLQ
		msg.Nack(false, false)
		return
	}

	// --- Idempotency check ---
	if event.EventID == "" {
		log.Printf("[Consumer] Message missing event_id, rejecting")
		msg.Nack(false, false)
		return
	}

	seen, err := c.store.IsSeen(event.EventID)
	if err != nil {
		log.Printf("[Consumer] Idempotency check error: %v — requeuing", err)
		msg.Nack(false, true) // requeue; transient DB error
		return
	}

	if seen {
		log.Printf("[Consumer] Duplicate event %s — skipping (idempotent)", event.EventID)
		msg.Ack(false) // ACK to remove from queue without processing
		return
	}

	// --- Simulate sending email ---
	if err := sendNotification(event); err != nil {
		log.Printf("[Consumer] Failed to send notification: %v — checking retry count", err)
		retryCount := getRetryCount(msg)
		if retryCount >= broker.MaxRetries {
			log.Printf("[Consumer] Max retries (%d) reached for event %s — sending to DLQ", broker.MaxRetries, event.EventID)
			msg.Nack(false, false) // reject to DLQ
		} else {
			log.Printf("[Consumer] Retry %d/%d for event %s", retryCount+1, broker.MaxRetries, event.EventID)
			msg.Nack(false, false) // reject without requeue (DLX handles retry via TTL for demo)
		}
		return
	}

	// --- Mark as processed BEFORE ACKing (at-least-once guarantee) ---
	if err := c.store.MarkSeen(event.EventID); err != nil {
		log.Printf("[Consumer] Failed to mark event %s as seen: %v — requeuing", event.EventID, err)
		msg.Nack(false, true)
		return
	}

	// --- ACK only after successful processing and idempotency record ---
	msg.Ack(false)
}

// sendNotification simulates sending an email by logging to the console.
func sendNotification(event domain.PaymentEvent) error {
	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f Status: %s",
		event.CustomerEmail,
		event.OrderID,
		float64(event.Amount)/100.0,
		event.Status,
	)
	return nil
}

// getRetryCount reads the x-death header to determine how many times a message was requeued.
func getRetryCount(msg amqp.Delivery) int {
	if msg.Headers == nil {
		return 0
	}
	deaths, ok := msg.Headers["x-death"]
	if !ok {
		return 0
	}
	deathList, ok := deaths.([]interface{})
	if !ok || len(deathList) == 0 {
		return 0
	}
	first, ok := deathList[0].(amqp.Table)
	if !ok {
		return 0
	}
	count, ok := first["count"].(int64)
	if !ok {
		return 0
	}
	return int(count)
}
