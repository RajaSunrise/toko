package handlers

import (
	"fmt"
	"log"
	"toko/internal/models"
	"toko/internal/services"

	"github.com/gofiber/fiber/v2"
)

// OrderHandler handles HTTP requests for orders.
type OrderHandler struct {
	service *services.OrderService
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(service *services.OrderService) *OrderHandler {
	return &OrderHandler{
		service: service,
	}
}

// RegisterRoutes registers the order routes with the Fiber app.
func (h *OrderHandler) RegisterRoutes(router fiber.Router) {
	orderRoutes := router.Group("/orders")
	orderRoutes.Get("/", h.HandleGetOrders)
	orderRoutes.Get("/:id", h.HandleGetOrderByID)
	orderRoutes.Post("/", h.HandleCreateOrder)
	// Example for updating order status, could be managed by admin or user role
	orderRoutes.Patch("/:id/status", h.HandleUpdateOrderStatus)
}

// HandleGetOrders retrieves all orders.
// In a real app, this would likely be filtered by user ID based on authentication context.
func (h *OrderHandler) HandleGetOrders(c *fiber.Ctx) error {
	orders, err := h.service.GetAllOrders()
	if err != nil {
		log.Printf("Error getting all orders: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not retrieve orders",
			"error":   err.Error(),
		})
	}
	return c.JSON(orders)
}

// HandleGetOrderByID retrieves a single order by its ID.
func (h *OrderHandler) HandleGetOrderByID(c *fiber.Ctx) error {
	orderID := c.Params("id")
	order, err := h.service.GetOrderByID(orderID)
	if err != nil {
		log.Printf("Error getting order by ID %s: %v", orderID, err)
		// Check if the error is because the order was not found
		if err.Error() == fmt.Sprintf("order with ID %s not found", orderID) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": fmt.Sprintf("Order with ID %s not found", orderID),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not retrieve order",
			"error":   err.Error(),
		})
	}
	return c.JSON(order)
}

// HandleCreateOrder creates a new order.
func (h *OrderHandler) HandleCreateOrder(c *fiber.Ctx) error {
	var orderRequest models.Order
	// Attempt to bind the request body to the Order model
	if err := c.BodyParser(&orderRequest); err != nil {
		log.Printf("Error parsing request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	// Basic validation: UserID and Items are required
	if orderRequest.UserID == "" || len(orderRequest.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "UserID and at least one item are required for an order.",
		})
	}

	// Add any additional validation for items if needed (e.g., quantity > 0)

	// Call the service to create the order. The service handles validation,
	// repository interaction, and RabbitMQ publishing.
	createdOrder, err := h.service.CreateOrder(orderRequest)
	if err != nil {
		log.Printf("Error creating order: %v", err)
		// Specific error handling based on service errors (e.g., insufficient stock)
		if err.Error() == "insufficient stock" { // Example error string
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Order creation failed due to insufficient stock.",
				"error":   err.Error(),
			})
		}
		// Generic error for other issues
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not create order",
			"error":   err.Error(),
		})
	}

	// Return the created order with its new ID and a 201 Created status
	return c.Status(fiber.StatusCreated).JSON(createdOrder)
}

// HandleUpdateOrderStatus updates the status of an existing order.
func (h *OrderHandler) HandleUpdateOrderStatus(c *fiber.Ctx) error {
	orderID := c.Params("id")
	var updateData struct {
		Status string `json:"status"`
	}

	if err := c.BodyParser(&updateData); err != nil {
		log.Printf("Error parsing request body for status update: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body for status update",
			"error":   err.Error(),
		})
	}

	if updateData.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Status is required for order status update.",
		})
	}

	err := h.service.UpdateOrderStatus(orderID, updateData.Status)
	if err != nil {
		log.Printf("Error updating order status for order %s: %v", orderID, err)
		// Check for specific errors like "order not found" or "invalid status"
		if err.Error() == fmt.Sprintf("order with ID %s not found for status update", orderID) ||
			err.Error() == fmt.Sprintf("invalid order status: %s", updateData.Status) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": fmt.Sprintf("Order update failed: %v", err.Error()),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not update order status",
			"error":   err.Error(),
		})
	}

	// Respond with a success message or the updated order if the service returned it
	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Order %s status updated successfully to %s", orderID, updateData.Status),
	})
}
