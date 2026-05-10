package provider

import (
	"fmt"
	"log"
	"net/smtp"
	"notification-service/internal/domain"
)

// SMTPProvider is a real email adapter using standard net/smtp.
// Configure via environment variables: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM.
type SMTPProvider struct {
	host string
	port string
	auth smtp.Auth
	from string
}

func NewSMTPProvider(host, port, user, pass, from string) *SMTPProvider {
	var auth smtp.Auth
	if user != "" && pass != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	return &SMTPProvider{host: host, port: port, auth: auth, from: from}
}

func (p *SMTPProvider) Send(event domain.PaymentEvent) error {
	addr := fmt.Sprintf("%s:%s", p.host, p.port)
	subject := fmt.Sprintf("Order #%s — %s", event.OrderID, event.Status)
	body := fmt.Sprintf(
		"Hello,\n\nYour order #%s has been updated.\nStatus: %s\nAmount: $%.2f\n\nThank you!",
		event.OrderID, event.Status, float64(event.Amount)/100.0,
	)
	msg := []byte(
		"To: " + event.CustomerEmail + "\r\n" +
			"From: " + p.from + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			body + "\r\n",
	)

	err := smtp.SendMail(addr, p.auth, p.from, []string{event.CustomerEmail}, msg)
	if err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	log.Printf("[SMTPProvider] ✉ Email sent to %s for order %s", event.CustomerEmail, event.OrderID)
	return nil
}
