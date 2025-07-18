package repositories

import (
	"toko/internal/models"
)

// ProductRepository defines the interface for product data access.
type ProductRepository interface {
	GetAll() ([]models.Product, error)
	GetByID(id string) (*models.Product, error)
	Create(product *models.Product) error
	Update(product *models.Product) error
	Delete(id string) error
}
