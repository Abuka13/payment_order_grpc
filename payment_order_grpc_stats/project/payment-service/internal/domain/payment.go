package domain

type Payment struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
}

// PaymentStats holds aggregated statistics across all payments.
type PaymentStats struct {
	TotalCount      int64
	AuthorizedCount int64
	DeclinedCount   int64
	TotalAmount     int64
}
