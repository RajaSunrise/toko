package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/streadway/amqp"

	"toko/internal/handlers"
	"toko/internal/models"
	"toko/internal/repositories"
	"toko/internal/services"
	"toko/pkg/rabbitmq"

	"github.com/spf13/viper"
)

func main() {
	// --- Configuration ---
	// Set up Viper to read configuration from environment variables or a file
	viper.SetDefault("APP_PORT", ":8080")
	viper.SetDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	viper.AutomaticEnv() // Load environment variables

	appPort := viper.GetString("APP_PORT")
	rabbitMQURL := viper.GetString("RABBITMQ_URL")

	// --- Initialize RabbitMQ Client ---
	// Note: The RabbitMQ client needs to be properly managed, especially for connections.
	// For simplicity, we initialize it here. In a larger app, consider a dedicated
	// package for managing these resources.
	mqConfig := rabbitmq.Config{URL: rabbitMQURL}
	mqClient, err := rabbitmq.NewClient(mqConfig)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ client: %v", err)
	}
	defer mqClient.Close() // Ensure the connection is closed on exit

	// --- Initialize Repositories (using mocks for now) ---
	productRepo := repositories.NewMockProductRepository()
	orderRepo := repositories.NewMockOrderRepository()

	// Seed some mock data for products
	seedProducts(productRepo)

	// --- Initialize Services ---
	// ProductService depends on ProductRepository
	productService := services.NewProductService(productRepo)
	// OrderService depends on OrderRepository, ProductRepository, and RabbitMQClient
	orderService := services.NewOrderService(orderRepo, productRepo, mqClient)

	// --- Initialize Handlers ---
	productHandler := handlers.NewProductHandler(productService)
	orderHandler := handlers.NewOrderHandler(orderService)

	// --- Initialize Fiber App ---
	app := fiber.New()

	// --- Middleware ---
	app.Use(logger.New()) // Request logger

	// --- API Routes ---
	// Group routes under /api/v1
	apiV1 := app.Group("/api/v1")

	// Register product routes
	productHandler.RegisterRoutes(apiV1)
	// Register order routes
	orderHandler.RegisterRoutes(apiV1)

	// --- Health Check Endpoint ---
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "healthy",
			"time":    time.Now().Format(time.RFC3339),
			"rabbitM": "connected", // Simple check, real check needs MQ ping
		})
	})

	// --- Start RabbitMQ Consumer in a Goroutine ---
	// This consumer will listen for order-related events.
	// In a real app, you'd have more sophisticated error handling,
	// connection recovery, and message processing logic.
	go func() {
		log.Println("Starting RabbitMQ consumer for orders...")
		// Define how to handle incoming order messages
		messageHandler := func(msg amqp.Delivery) error {
			log.Printf("Received Order Event (Tag: %d): %s", msg.DeliveryTag, string(msg.Body))
			// Here, you would parse the message (e.g., order details)
			// and trigger business logic (e.g., update inventory, send email).
			// For this example, we'll just simulate work.
			time.Sleep(1 * time.Second) // Simulate processing time

			// Manually acknowledge the message if processing was successful
			// If an error occurs during processing, Nack the message to requeue or send to dead-letter queue
			return nil // Return nil to acknowledge
		}
		if consumerErr := mqClient.ConsumeOrderEvents(messageHandler); consumerErr != nil {
			log.Printf("Failed to start RabbitMQ consumer: %v", err)
			// In a production system, you'd want to implement reconnection logic
		}
	}()

	// --- Start HTTP Server ---
	log.Printf("Starting server on port %s", appPort)

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(appPort); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	<-quit
	log.Println("Shutting down server...")

	// Shutdown Fiber app
	if err := app.Shutdown(); err != nil {
		log.Printf("Error during Fiber shutdown: %v", err)
	}

	// Close RabbitMQ connection is handled by defer in main
	log.Println("Server gracefully stopped")
}

// seedProducts populates the mock product repository with some initial data.
func seedProducts(repo repositories.ProductRepository) {
	products := []models.Product{
		{ID: "prod-1", Name: "Laptop", Description: "High performance laptop", Price: 1200.00, Stock: 10},
		{ID: "prod-2", Name: "Keyboard", Description: "Mechanical keyboard", Price: 75.00, Stock: 25},
		{ID: "prod-3", Name: "Mouse", Description: "Ergonomic wireless mouse", Price: 25.00, Stock: 50},
	}

	for i := range products {
		// Using mock repository's Create, which handles ID generation if not provided
		// For seeding, we explicitly set IDs to ensure consistency.
		err := repo.Create(&products[i])
		if err != nil {
			log.Printf("Error seeding product %s: %v", products[i].Name, err)
		} else {
			log.Printf("Seeded product: %s (ID: %s)", products[i].Name, products[i].ID)
		}
	}
}
