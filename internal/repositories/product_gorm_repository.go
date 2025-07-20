package repositories

import (
	"fmt"
	"toko/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GORMProductRepository is a GORM implementation of ProductRepository.
type GORMProductRepository struct {
	db *gorm.DB
}

// NewGORMProductRepository creates a new instance of GORMProductRepository.
func NewGORMProductRepository(db *gorm.DB) *GORMProductRepository {
	return &GORMProductRepository{
		db: db,
	}
}

// GetAll retrieves all products from the database.
func (r *GORMProductRepository) GetAll() ([]models.Product, error) {
	var products []models.Product
	if err := r.db.Find(&products).Error; err != nil {
		return nil, fmt.Errorf("failed to get all products: %w", err)
	}
	return products, nil
}

// GetByID retrieves a single product by its ID from the database.
func (r *GORMProductRepository) GetByID(id string) (*models.Product, error) {
	var product models.Product
	if err := r.db.First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("product with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get product by ID %s: %w", id, err)
	}
	return &product, nil
}

// Create creates a new product in the database.
func (r *GORMProductRepository) Create(product *models.Product) error {
	if product.ID == "" {
		product.ID = uuid.New().String()
	}
	if err := r.db.Create(product).Error; err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}
	return nil
}

// Update updates an existing product in the database.
func (r *GORMProductRepository) Update(product *models.Product) error {
	res := r.db.Save(product) // Save will update all fields, including zero values
	if res.Error != nil {
		return fmt.Errorf("failed to update product: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		// This case happens if the record doesn't exist.
		// GORM's Save doesn't return ErrRecordNotFound if no rows affected
		// for an update, so we check RowsAffected.
		return fmt.Errorf("product with ID %s not found for update", product.ID)
	}
	return nil
}

// Delete deletes a product by its ID from the database.
func (r *GORMProductRepository) Delete(id string) error {
	res := r.db.Delete(&models.Product{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("failed to delete product: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("product with ID %s not found for deletion", id)
	}
	return nil
}
