package usecase

import (
	"payment-service/internal/domain"
	"payment-service/internal/repository"

	"github.com/google/uuid"
)

type PaymentUsecase struct {
	repo repository.PaymentRepository
}

func NewPaymentUsecase(repo repository.PaymentRepository) *PaymentUsecase {
	return &PaymentUsecase{repo: repo}
}

func (u *PaymentUsecase) Create(orderID string, amount int64) (*domain.Payment, error) {
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
	}

	if err := u.repo.Create(payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (u *PaymentUsecase) GetByOrderID(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}
