package repositories

import (
	"toko/internal/models"
)

// OrderRepository defines the interface for order data access.
type OrderRepository interface {
	GetAll() ([]models.Order, error)
	GetByID(id string) (*models.Order, error)
	Create(order *models.Order) error
	UpdateStatus(id string, status string) error
	// Delete(id string) error // Deletion of orders might be complex, so we'll omit for now.
}
