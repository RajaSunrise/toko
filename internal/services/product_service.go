package services

import (
	"toko/internal/models"
	"toko/internal/repositories"
)

// ProductService handles business logic related to products.
type ProductService struct {
	repo repositories.ProductRepository
}

// NewProductService creates a new ProductService.
func NewProductService(repo repositories.ProductRepository) *ProductService {
	return &ProductService{
		repo: repo,
	}
}

// GetAllProducts retrieves all products.
func (s *ProductService) GetAllProducts() ([]models.Product, error) {
	return s.repo.GetAll()
}

// GetProductByID retrieves a single product by its ID.
func (s *ProductService) GetProductByID(id string) (*models.Product, error) {
	return s.repo.GetByID(id)
}

// CreateProduct creates a new product.
func (s *ProductService) CreateProduct(product *models.Product) error {
	// Add any business logic here, e.g., validation, default values.
	// For now, we'll just pass it to the repository.
	return s.repo.Create(product)
}

// UpdateProduct updates an existing product.
func (s *ProductService) UpdateProduct(product *models.Product) error {
	// Add any business logic here, e.g., validation.
	return s.repo.Update(product)
}

// DeleteProduct deletes a product by its ID.
func (s *ProductService) DeleteProduct(id string) error {
	return s.repo.Delete(id)
}
