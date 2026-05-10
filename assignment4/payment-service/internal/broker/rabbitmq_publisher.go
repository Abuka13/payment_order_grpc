package broker

import (
	"encoding/json"
	"fmt"
	"log"
	"payment-service/internal/domain"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queueName = "payment.completed"
	dlxName   = "payment.completed.dlx"
)

// PaymentEvent is the message payload published to the broker.
type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

// RabbitMQPublisher implements EventPublisher using RabbitMQ.
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQPublisher connects to RabbitMQ and declares the topology.
func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("[Publisher] Waiting for RabbitMQ... attempt %d: %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare DLX
	if err := ch.ExchangeDeclare(dlxName, "fanout", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare DLX: %w", err)
	}

	// Declare the main durable queue with DLX routing
	args := amqp.Table{
		"x-dead-letter-exchange": dlxName,
		"x-message-ttl":          int32(30000),
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, args); err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	// Enable publisher confirms for reliability
	if err := ch.Confirm(false); err != nil {
		return nil, fmt.Errorf("enable confirms: %w", err)
	}

	log.Println("[Publisher] Connected to RabbitMQ")
	return &RabbitMQPublisher{conn: conn, channel: ch}, nil
}

// PublishPaymentCompleted publishes a PaymentEvent after a successful payment.
// Uses publisher confirms to guarantee the message reached the broker.
func (p *RabbitMQPublisher) PublishPaymentCompleted(payment *domain.Payment) error {
	event := PaymentEvent{
		EventID:       payment.TransactionID, // TransactionID is unique per payment — perfect idempotency key
		OrderID:       payment.OrderID,
		Amount:        payment.Amount,
		CustomerEmail: payment.CustomerEmail,
		Status:        payment.Status,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	confirms := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	err = p.channel.Publish(
		"",        // default exchange
		queueName, // routing key = queue name
		true,      // mandatory: return if no queue binds
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // survive broker restart
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	// Wait for broker confirmation
	confirmed := <-confirms
	if !confirmed.Ack {
		return fmt.Errorf("broker did not ACK published message (event_id: %s)", event.EventID)
	}

	log.Printf("[Publisher] Event published: order_id=%s event_id=%s", payment.OrderID, event.EventID)
	return nil
}

// Close gracefully shuts down the publisher's channel and connection.
func (p *RabbitMQPublisher) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	log.Println("[Publisher] Connection closed")
}
