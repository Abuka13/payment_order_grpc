package broker

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QueueName    = "payment.completed"
	ExchangeName = ""
	DLXName      = "payment.completed.dlx"
	DLQName      = "payment.completed.dead"
	MaxRetries   = 3
)

// Connection wraps an AMQP connection and channel.
type Connection struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

// Connect establishes a connection to RabbitMQ with retries.
func Connect(url string) (*Connection, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("[Broker] Waiting for RabbitMQ... attempt %d: %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ after retries: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := setupTopology(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to setup topology: %w", err)
	}

	log.Println("[Broker] Connected to RabbitMQ and topology configured")
	return &Connection{conn: conn, Channel: ch}, nil
}

// setupTopology declares the Dead Letter Exchange, DLQ, and main durable queue.
func setupTopology(ch *amqp.Channel) error {
	// 1. Declare Dead Letter Exchange (fanout)
	if err := ch.ExchangeDeclare(
		DLXName, "fanout", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare DLX: %w", err)
	}

	// 2. Declare Dead Letter Queue
	if _, err := ch.QueueDeclare(
		DLQName, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare DLQ: %w", err)
	}

	// 3. Bind DLQ to DLX
	if err := ch.QueueBind(DLQName, "", DLXName, false, nil); err != nil {
		return fmt.Errorf("bind DLQ to DLX: %w", err)
	}

	// 4. Declare main queue with DLX routing and max-retries via x-death
	args := amqp.Table{
		"x-dead-letter-exchange": DLXName,
		"x-message-ttl":          int32(30000), // messages older than 30s go to DLQ on rejection
	}
	if _, err := ch.QueueDeclare(
		QueueName, true, false, false, false, args,
	); err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}

	// 5. Set QoS — process one message at a time for reliable ACKing
	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("set QoS: %w", err)
	}

	return nil
}

// Close gracefully shuts down the channel and connection.
func (c *Connection) Close() {
	if c.Channel != nil {
		c.Channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	log.Println("[Broker] Connection closed")
}
