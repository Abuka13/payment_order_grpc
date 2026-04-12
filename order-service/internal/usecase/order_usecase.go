package usecase

import (
	"errors"
	"order-service/internal/domain"
	"order-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

type PaymentClient interface {
	CreatePayment(orderID string, amount int64) (status string, transactionID string, err error)
}

type OrderUsecase struct {
	repo          repository.OrderRepository
	paymentClient PaymentClient
}

func NewOrderUsecase(repo repository.OrderRepository, paymentClient PaymentClient) *OrderUsecase {
	return &OrderUsecase{
		repo:          repo,
		paymentClient: paymentClient,
	}
}

func (u *OrderUsecase) Create(customerID, itemName string, amount int64) (*domain.Order, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	order := &domain.Order{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     "Pending",
		CreatedAt:  time.Now(),
	}

	if err := u.repo.Create(order); err != nil {
		return nil, err
	}

	status, _, err := u.paymentClient.CreatePayment(order.ID, order.Amount)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, "Failed")
		return nil, err
	}

	if status == "Authorized" {
		order.Status = "Paid"
		if err := u.repo.UpdateStatus(order.ID, "Paid"); err != nil {
			return nil, err
		}
	} else {
		order.Status = "Failed"
		if err := u.repo.UpdateStatus(order.ID, "Failed"); err != nil {
			return nil, err
		}
	}

	return order, nil
}

func (u *OrderUsecase) GetByID(id string) (*domain.Order, error) {
	return u.repo.GetByID(id)
}

func (u *OrderUsecase) Cancel(id string) error {
	order, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}

	if order.Status != "Pending" {
		return errors.New("only pending orders can be cancelled")
	}

	return u.repo.UpdateStatus(id, "Cancelled")
}
