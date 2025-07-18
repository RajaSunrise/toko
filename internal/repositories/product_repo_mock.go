package repositories

import (
	"fmt"
	"sync"
	"toko/internal/models"

	"github.com/google/uuid"
)

// MockProductRepository is an in-memory implementation of ProductRepository.
type MockProductRepository struct {
	products map[string]models.Product
	mu       sync.RWMutex
}

// NewMockProductRepository creates a new instance of MockProductRepository.
func NewMockProductRepository() *MockProductRepository {
	return &MockProductRepository{
		products: make(map[string]models.Product),
	}
}

// GetAll returns all products.
func (r *MockProductRepository) GetAll() ([]models.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	productList := make([]models.Product, 0, len(r.products))
	for _, p := range r.products {
		productList = append(productList, p)
	}
	return productList, nil
}

// GetByID returns a product by its ID.
func (r *MockProductRepository) GetByID(id string) (*models.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		return nil, fmt.Errorf("product with ID %s not found", id)
	}
	return &product, nil
}

// Create adds a new product.
func (r *MockProductRepository) Create(product *models.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if product.ID == "" {
		product.ID = uuid.New().String()
	}
	r.products[product.ID] = *product
	return nil
}

// Update modifies an existing product.
func (r *MockProductRepository) Update(product *models.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.products[product.ID]
	if !ok {
		return fmt.Errorf("product with ID %s not found for update", product.ID)
	}
	r.products[product.ID] = *product
	return nil
}

// Delete removes a product by its ID.
func (r *MockProductRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.products[id]
	if !ok {
		return fmt.Errorf("product with ID %s not found for deletion", id)
	}
	delete(r.products, id)
	return nil
}
