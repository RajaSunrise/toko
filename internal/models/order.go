package models

import "time"

// OrderItem represents a single item within an order.
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"` // Price at the time of order
}

// Order represents a customer order.
type Order struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	Status      string      `json:"status"` // e.g., "pending", "processing", "shipped", "delivered", "cancelled"
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}
