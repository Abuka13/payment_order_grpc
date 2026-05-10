package usecase

import (
	"log"
	"payment-service/internal/broker"
	"payment-service/internal/domain"
	"payment-service/internal/repository"

	"github.com/google/uuid"
)

type PaymentUsecase struct {
	repo      repository.PaymentRepository
	publisher broker.EventPublisher
}

func NewPaymentUsecase(repo repository.PaymentRepository, publisher broker.EventPublisher) *PaymentUsecase {
	return &PaymentUsecase{repo: repo, publisher: publisher}
}

func (u *PaymentUsecase) Create(orderID string, amount int64, customerEmail string) (*domain.Payment, error) {
	status := "Authorized"
	if amount > 100000 {
		status = "Declined"
	}

	payment := &domain.Payment{
		ID:            uuid.NewString(),
		OrderID:       orderID,
		TransactionID: uuid.NewString(),
		Amount:        amount,
		Status:        status,
		CustomerEmail: customerEmail,
	}

	if err := u.repo.Create(payment); err != nil {
		return nil, err
	}

	// Publish event AFTER successful DB commit (at-least-once delivery)
	if status == "Authorized" {
		if err := u.publisher.PublishPaymentCompleted(payment); err != nil {
			log.Printf("[Usecase] Warning: failed to publish event for order %s: %v", orderID, err)
		}
	}

	return payment, nil
}

func (u *PaymentUsecase) GetByOrderID(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}
