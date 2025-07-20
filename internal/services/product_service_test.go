package services_test

import (
	"fmt"
	"testing"

	"toko/internal/models"
	"toko/internal/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProductRepository is a mock implementation of repositories.ProductRepository
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) GetAll() ([]models.Product, error) {
	args := m.Called()
	return args.Get(0).([]models.Product), args.Error(1)
}

func (m *MockProductRepository) GetByID(id string) (*models.Product, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductRepository) Create(product *models.Product) error {
	args := m.Called(product)
	return args.Error(0)
}

func (m *MockProductRepository) Update(product *models.Product) error {
	args := m.Called(product)
	return args.Error(0)
}

func (m *MockProductRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestProductService_GetAllProducts(t *testing.T) {
	mockRepo := new(MockProductRepository)
	service := services.NewProductService(mockRepo)

	expectedProducts := []models.Product{
		{ID: "1", Name: "Product A", Price: 10.0, Stock: 100},
		{ID: "2", Name: "Product B", Price: 20.0, Stock: 50},
	}

	mockRepo.On("GetAll").Return(expectedProducts, nil).Once()

	products, err := service.GetAllProducts()

	assert.NoError(t, err)
	assert.Len(t, products, 2)
	assert.Equal(t, expectedProducts, products)
	mockRepo.AssertExpectations(t)
}

func TestProductService_GetProductByID(t *testing.T) {
	mockRepo := new(MockProductRepository)
	service := services.NewProductService(mockRepo)

	expectedProduct := &models.Product{ID: "1", Name: "Product A", Price: 10.0, Stock: 100}

	// Test successful retrieval
	mockRepo.On("GetByID", "1").Return(expectedProduct, nil).Once()
	product, err := service.GetProductByID("1")
	assert.NoError(t, err)
	assert.Equal(t, expectedProduct, product)
	mockRepo.AssertExpectations(t)

	// Test product not found
	mockRepo.On("GetByID", "99").Return(nil, fmt.Errorf("product with ID 99 not found")).Once()
	product, err = service.GetProductByID("99")
	assert.Error(t, err)
	assert.Nil(t, product)
	assert.Contains(t, err.Error(), "not found")
	mockRepo.AssertExpectations(t)
}

func TestProductService_CreateProduct(t *testing.T) {
	mockRepo := new(MockProductRepository)
	service := services.NewProductService(mockRepo)

	newProduct := &models.Product{Name: "New Product", Price: 50.0, Stock: 20}

	// Test successful creation
	mockRepo.On("Create", newProduct).Return(nil).Once()
	err := service.CreateProduct(newProduct)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Test creation failure (e.g., database error)
	mockRepo.On("Create", newProduct).Return(fmt.Errorf("database error")).Once()
	err = service.CreateProduct(newProduct)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
	mockRepo.AssertExpectations(t)
}

func TestProductService_UpdateProduct(t *testing.T) {
	mockRepo := new(MockProductRepository)
	service := services.NewProductService(mockRepo)

	updatedProduct := &models.Product{ID: "1", Name: "Product A Updated", Price: 12.0, Stock: 95}

	// Test successful update
	mockRepo.On("Update", updatedProduct).Return(nil).Once()
	err := service.UpdateProduct(updatedProduct)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Test update failure (e.g., product not found in repo)
	mockRepo.On("Update", &models.Product{ID: "99", Name: "NonExistent", Price: 1.0, Stock: 1}).Return(fmt.Errorf("product with ID 99 not found for update")).Once()
	err = service.UpdateProduct(&models.Product{ID: "99", Name: "NonExistent", Price: 1.0, Stock: 1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found for update")
	mockRepo.AssertExpectations(t)
}

func TestProductService_DeleteProduct(t *testing.T) {
	mockRepo := new(MockProductRepository)
	service := services.NewProductService(mockRepo)

	// Test successful deletion
	mockRepo.On("Delete", "1").Return(nil).Once()
	err := service.DeleteProduct("1")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Test deletion failure (e.g., product not found)
	mockRepo.On("Delete", "99").Return(fmt.Errorf("product with ID 99 not found for deletion")).Once()
	err = service.DeleteProduct("99")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found for deletion")
	mockRepo.AssertExpectations(t)
}
