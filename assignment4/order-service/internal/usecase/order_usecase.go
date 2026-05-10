package usecase

import (
	"context"
	"errors"
	"order-service/internal/domain"
	"order-service/internal/repository"
	"time"

	"github.com/google/uuid"
)

type PaymentClient interface {
	CreatePayment(orderID string, amount int64, customerEmail string) (status string, transactionID string, err error)
}

// OrderCache defines the caching contract used by the use case.
// The use case knows nothing about Redis — only this interface.
type OrderCache interface {
	Get(ctx context.Context, id string) (*domain.Order, error)
	Set(ctx context.Context, order *domain.Order) error
	Invalidate(ctx context.Context, id string)
}

type OrderUsecase struct {
	repo          repository.OrderRepository
	paymentClient PaymentClient
	cache         OrderCache
}

func NewOrderUsecase(repo repository.OrderRepository, paymentClient PaymentClient, cache OrderCache) *OrderUsecase {
	return &OrderUsecase{
		repo:          repo,
		paymentClient: paymentClient,
		cache:         cache,
	}
}

func (u *OrderUsecase) Create(customerID, itemName string, amount int64, customerEmail string) (*domain.Order, error) {
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

	status, _, err := u.paymentClient.CreatePayment(order.ID, order.Amount, customerEmail)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, "Failed")
		u.cache.Invalidate(context.Background(), order.ID)
		return nil, err
	}

	if status == "Authorized" {
		order.Status = "Paid"
	} else {
		order.Status = "Failed"
	}

	if err := u.repo.UpdateStatus(order.ID, order.Status); err != nil {
		return nil, err
	}
	// Atomic invalidation: immediately delete stale cache entry after DB update
	u.cache.Invalidate(context.Background(), order.ID)

	return order, nil
}

// GetByID implements cache-aside: check cache first, fall back to DB.
func (u *OrderUsecase) GetByID(id string) (*domain.Order, error) {
	ctx := context.Background()

	// 1. Check cache
	cached, err := u.cache.Get(ctx, id)
	if err == nil && cached != nil {
		return cached, nil
	}

	// 2. Cache miss — query DB
	order, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 3. Populate cache (best-effort, ignore error)
	_ = u.cache.Set(ctx, order)

	return order, nil
}

func (u *OrderUsecase) Cancel(id string) error {
	order, err := u.repo.GetByID(id)
	if err != nil {
		return err
	}

	if order.Status != "Pending" {
		return errors.New("only pending orders can be cancelled")
	}

	if err := u.repo.UpdateStatus(id, "Cancelled"); err != nil {
		return err
	}
	// Invalidate cache after status change
	u.cache.Invalidate(context.Background(), id)
	return nil
}
