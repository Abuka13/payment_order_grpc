package postgres

import (
	"database/sql"
	"errors"
	"payment-service/internal/domain"
	"payment-service/internal/repository"
)

type PaymentPostgresRepository struct {
	db *sql.DB
}

func NewPaymentPostgresRepository(db *sql.DB) repository.PaymentRepository {
	return &PaymentPostgresRepository{db: db}
}

func (r *PaymentPostgresRepository) Create(payment *domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, transaction_id, amount, status, customer_email)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query,
		payment.ID,
		payment.OrderID,
		payment.TransactionID,
		payment.Amount,
		payment.Status,
		payment.CustomerEmail,
	)
	return err
}

func (r *PaymentPostgresRepository) GetByOrderID(orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, transaction_id, amount, status, customer_email
		FROM payments
		WHERE order_id = $1
	`

	var payment domain.Payment
	err := r.db.QueryRow(query, orderID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.TransactionID,
		&payment.Amount,
		&payment.Status,
		&payment.CustomerEmail,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &payment, nil
}
