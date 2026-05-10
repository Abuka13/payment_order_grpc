package broker

import "payment-service/internal/domain"

// EventPublisher defines the contract for publishing payment events.
// Hiding the broker implementation behind this interface keeps
// the usecase layer decoupled from RabbitMQ details.
type EventPublisher interface {
	PublishPaymentCompleted(payment *domain.Payment) error
	Close()
}
