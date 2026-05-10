package provider

import "notification-service/internal/domain"

// EmailSender is the port that the consumer depends on.
// Business logic never imports a concrete SMTP or Mailjet client.
type EmailSender interface {
	Send(event domain.PaymentEvent) error
}
