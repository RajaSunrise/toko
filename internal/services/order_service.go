package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	"toko/internal/models"
	"toko/internal/repositories"
	"toko/pkg/rabbitmq"

	"github.com/google/uuid"
)

// OrderService handles business logic related to orders.
type OrderService struct {
	orderRepo   repositories.OrderRepository
	productRepo repositories.ProductRepository
	mqClient    *rabbitmq.Client // RabbitMQ client
}

// NewOrderService creates a new OrderService.
func NewOrderService(orderRepo repositories.OrderRepository, productRepo repositories.ProductRepository, mqClient *rabbitmq.Client) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		mqClient:    mqClient,
	}
}

// GetAllOrders retrieves all orders.
func (s *OrderService) GetAllOrders() ([]models.Order, error) {
	return s.orderRepo.GetAll()
}

// GetOrderByID retrieves a single order by its ID.
func (s *OrderService) GetOrderByID(id string) (*models.Order, error) {
	return s.orderRepo.GetByID(id)
}

// CreateOrder creates a new order.
func (s *OrderService) CreateOrder(orderRequest models.Order) (*models.Order, error) {
	// 1. Validate products and calculate total amount
	var totalAmount float64
	var processedItems []models.OrderItem

	// Start a transaction if using a real DB. For mock, we simulate atomicity.
	for _, item := range orderRequest.Items {
		product, err := s.productRepo.GetByID(item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %s not found: %w", item.ProductID, err)
		}

		// In a real scenario, you'd check stock here.
		// For mock, we assume stock is sufficient or handled elsewhere.
		if product.Stock < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for product %s (requested: %d, available: %d)", product.Name, item.Quantity, product.Stock)
		}

		itemPrice := product.Price // Use price at the time of order creation
		processedItems = append(processedItems, models.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     itemPrice,
		})
		totalAmount += itemPrice * float64(item.Quantity)
	}

	// Create the order object
	newOrder := &models.Order{
		ID:          uuid.New().String(),
		UserID:      orderRequest.UserID, // Assuming UserID is provided or derived from auth context
		Items:       processedItems,
		TotalAmount: totalAmount,
		Status:      "pending", // Initial status
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 2. Save the order to the repository
	err := s.orderRepo.Create(newOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to create order in repository: %w", err)
	}

	// 3. Publish an event to RabbitMQ for order creation
	// This could be an "order.created" event.
	// The message should contain enough info for consumers to process it.
	orderCreatedMessage := map[string]interface{}{
		"orderID": newOrder.ID,
		"userID":  newOrder.UserID,
		"status":  newOrder.Status,
		"total":   newOrder.TotalAmount,
		// Include items if needed by consumers
	}

	// Use the RabbitMQ client to publish the message
	if s.mqClient != nil {
		messageBody, err := json.Marshal(orderCreatedMessage)
		if err != nil {
			log.Printf("Failed to marshal order to JSON: %v", err)
		} else {
			err = s.mqClient.Publish("order", "order.created", messageBody)
			if err != nil {
				log.Printf("Warning: Failed to publish order created event for order %s: %v", newOrder.ID, err)
			} else {
				log.Printf("Successfully published order created event for order %s", newOrder.ID)
			}
		}

	} else {
		log.Println("RabbitMQ client is not initialized. Skipping message publication.")
	}

	return newOrder, nil
}

// UpdateOrderStatus updates the status of an existing order.
func (s *OrderService) UpdateOrderStatus(id string, status string) error {
	// Add validation for status if necessary
	validStatuses := map[string]bool{"pending": true, "processing": true, "shipped": true, "delivered": true, "cancelled": true}
	if _, ok := validStatuses[status]; !ok {
		return fmt.Errorf("invalid order status: %s", status)
	}

	err := s.orderRepo.UpdateStatus(id, status)
	if err != nil {
		return fmt.Errorf("failed to update order status for order %s: %w", id, err)
	}

	// Optionally, publish an event for order status update
	// err = s.rabbitMQClient.PublishOrderStatusUpdated(id, status)
	// if err != nil {
	// 	log.Printf("Warning: Failed to publish order status update event for order %s: %v", id, err)
	// }

	return nil
}
