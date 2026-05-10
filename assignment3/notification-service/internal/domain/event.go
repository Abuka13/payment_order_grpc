package domain

// PaymentEvent is the message published to the broker after a successful payment.
type PaymentEvent struct {
	EventID       string  `json:"event_id"`        // Unique event ID (used for idempotency)
	OrderID       string  `json:"order_id"`
	Amount        int64   `json:"amount"`
	CustomerEmail string  `json:"customer_email"`
	Status        string  `json:"status"`
}
