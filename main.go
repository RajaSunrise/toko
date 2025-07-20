package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"toko/internal/handlers"
	"toko/internal/middleware"
	"toko/internal/models"
	"toko/internal/repositories"
	"toko/internal/services"
	"toko/pkg/rabbitmq"
)

// NewApp creates and configures the Fiber application.
// This function is designed to be callable from tests.
func NewApp() (*fiber.App, *services.AuthService, error) {
	// --- Configuration ---
	// Set up Viper to read configuration from environment variables or a file
	viper.SetDefault("APP_PORT", ":8080")
	viper.SetDefault("DATABASE_DSN", "host=127.0.0.1 user=postgres password=postgres dbname=toko port=5432 sslmode=disable")
	viper.SetDefault("JWT_SECRET", "supersecretjwtkey")
	viper.SetDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	viper.AutomaticEnv() // Load environment variables

	databaseDSN := viper.GetString("DATABASE_DSN")
	jwtSecret := viper.GetString("JWT_SECRET")
	rabbitMQURL := viper.GetString("RABBITMQ_URL")

	// --- Initialize Database (GORM) ---
	db, err := gorm.Open(postgres.Open(databaseDSN), &gorm.Config{}) // Use postgres.Open
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate database schema
	err = db.AutoMigrate(&models.Product{}, &models.User{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	// --- Initialize Repositories (using GORM) ---
	productRepo := repositories.NewGORMProductRepository(db)
	userRepo := repositories.NewGORMUserRepository(db)
	orderRepo := repositories.NewMockOrderRepository() // Keep mock for now

	// --- Initialize RabbitMQ Client ---
	mqConfig := rabbitmq.Config{URL: rabbitMQURL}
	mqClient, err := rabbitmq.NewClient(mqConfig)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ client: %v", err)
	}
	defer mqClient.Close()

	// --- Initialize Services ---
	productService := services.NewProductService(productRepo)
	orderService := services.NewOrderService(orderRepo, productRepo, mqClient)
	authService := services.NewAuthService(userRepo, jwtSecret)

	// --- Initialize Handlers ---
	productHandler := handlers.NewProductHandler(productService)
	orderHandler := handlers.NewOrderHandler(orderService)
	authHandler := handlers.NewAuthHandler(authService)

	// --- Initialize Fiber App ---
	app := fiber.New()

	// --- Middleware ---
	app.Use(logger.New()) // Request logger

	// --- API Routes ---
	// Group routes under /api/v1
	apiV1 := app.Group("/api/v1")

	// Authentication routes (public)
	authHandler.RegisterRoutes(apiV1)

	// Protected routes (require JWT authentication)
	protectedRoutes := apiV1.Group("", middleware.AuthRequired(authService))

	// Register product routes
	productHandler.RegisterRoutes(protectedRoutes)
	// Register order routes
	orderHandler.RegisterRoutes(protectedRoutes)

	// --- Health Check Endpoint ---
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
			// "rabbitM": "connected", // Simple check, real check needs MQ ping
		})
	})

	return app, authService, nil
}

// main is the entry point of the application.
func main() {
	app, _, err := NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Retrieve the configured port, default to 8080 if not set
	appPort := viper.GetString("APP_PORT")
	if appPort == "" {
		appPort = ":8080"
	}

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

	// Note: If RabbitMQ was active, its connection would also need to be closed here.
	log.Println("Server gracefully stopped")
}

// seedProducts populates the product repository with some initial data.
func seedProducts(repo repositories.ProductRepository) {
	products := []models.Product{
		{Name: "Laptop", Description: "High performance laptop", Price: 1200.00, Stock: 10},
		{Name: "Keyboard", Description: "Mechanical keyboard", Price: 75.00, Stock: 25},
		{Name: "Mouse", Description: "Ergonomic wireless mouse", Price: 25.00, Stock: 50},
	}

	for _, product := range products {
		if err := repo.Create(&product); err != nil {
			log.Printf("Error seeding product: %v", err)
		}
	}
}
