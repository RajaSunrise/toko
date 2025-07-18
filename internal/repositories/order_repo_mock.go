package repositories

import (
	"fmt"
	"sync"
	"time"
	"toko/internal/models"

	"github.com/google/uuid"
)

// MockOrderRepository is an in-memory implementation of OrderRepository.
type MockOrderRepository struct {
	orders map[string]models.Order
	mu     sync.RWMutex
}

// NewMockOrderRepository creates a new instance of MockOrderRepository.
func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{
		orders: make(map[string]models.Order),
	}
}

// GetAll returns all orders.
func (r *MockOrderRepository) GetAll() ([]models.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orderList := make([]models.Order, 0, len(r.orders))
	for _, order := range r.orders {
		orderList = append(orderList, order)
	}
	return orderList, nil
}

// GetByID returns an order by its ID.
func (r *MockOrderRepository) GetByID(id string) (*models.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, ok := r.orders[id]
	if !ok {
		return nil, fmt.Errorf("order with ID %s not found", id)
	}
	return &order, nil
}

// Create adds a new order.
func (r *MockOrderRepository) Create(order *models.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if order.ID == "" {
		order.ID = uuid.New().String()
	}
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	r.orders[order.ID] = *order
	return nil
}

// UpdateStatus updates the status of an order.
func (r *MockOrderRepository) UpdateStatus(id string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	order, ok := r.orders[id]
	if !ok {
		return fmt.Errorf("order with ID %s not found for status update", id)
	}
	order.Status = status
	order.UpdatedAt = time.Now()
	r.orders[id] = order
	return nil
}

