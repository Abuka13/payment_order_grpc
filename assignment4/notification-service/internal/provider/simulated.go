package provider

import (
	"errors"
	"log"
	"math/rand"
	"notification-service/internal/domain"
	"time"
)

// SimulatedProvider logs the email action and mimics real-world conditions:
//   - network latency via time.Sleep
//   - occasional random failures (20 % failure rate) to exercise retry logic
type SimulatedProvider struct {
	failureRate float64 // 0.0 – 1.0
}

func NewSimulatedProvider(failureRate float64) *SimulatedProvider {
	return &SimulatedProvider{failureRate: failureRate}
}

func (p *SimulatedProvider) Send(event domain.PaymentEvent) error {
	// Simulate network latency (50–200 ms)
	latency := time.Duration(50+rand.Intn(150)) * time.Millisecond
	time.Sleep(latency)

	// Simulate occasional failure
	if rand.Float64() < p.failureRate {
		return errors.New("simulated provider: transient network error")
	}

	log.Printf("[SimulatedProvider] ✉ Email sent to %s | Order: %s | Amount: $%.2f | Status: %s",
		event.CustomerEmail,
		event.OrderID,
		float64(event.Amount)/100.0,
		event.Status,
	)
	return nil
}
