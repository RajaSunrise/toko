package main_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres" // Use PostgreSQL driver
	"gorm.io/gorm"

	mainapp "toko" // Alias the main package for clarity
	"toko/internal/models"
	"toko/internal/repositories"
	"toko/internal/services"
)

// MockRabbitMQClient is a mock implementation of the RabbitMQ client
type MockRabbitMQClient struct {
	mock.Mock
}

func (m *MockRabbitMQClient) Publish(exchange, routingKey string, body []byte) error {
	args := m.Called(exchange, routingKey, body)
	return args.Error(0)
}

func (m *MockRabbitMQClient) PublishOrderCreated(messageBody map[string]interface{}) error {
	args := m.Called(messageBody)
	return args.Error(0)
}

func (m *MockRabbitMQClient) ConsumeOrderEvents(messageHandler func(amqp.Delivery) error) error {
	args := m.Called(messageHandler)
	return args.Error(0)
}

func (m *MockRabbitMQClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

var (
	v           *viper.Viper
	db          *gorm.DB
	app         *fiber.App
	authService *services.AuthService
	productRepo repositories.ProductRepository
	orderRepo   repositories.OrderRepository
	userRepo    repositories.UserRepository
	mockMQ      *MockRabbitMQClient
)

func TestMain(m *testing.M) {
	// Initialize Viper for tests
	v = viper.New()
	v.SetDefault("APP_PORT", ":8081") // Use a different port for tests
	v.SetDefault("DATABASE_DSN", "host=127.0.0.1 user=postgres password=postgres dbname=toko port=5432 sslmode=disable") // PostgreSQL DSN
	v.SetDefault("JWT_SECRET", "test_jwt_secret")
	v.SetDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	v.AutomaticEnv()

	// Initialize Database (GORM)
	var err error
	db, err = gorm.Open(postgres.Open(v.GetString("DATABASE_DSN")), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate models
	err = db.AutoMigrate(&models.Product{}, &models.User{})
	if err != nil {
		log.Fatalf("Failed to migrate test database: %v", err)
	}

	// Initialize Repositories
	productRepo = repositories.NewGORMProductRepository(db)
	orderRepo = repositories.NewMockOrderRepository() // Using mock for order for simplicity
	userRepo = repositories.NewGORMUserRepository(db)

	// Seed database for test
	seedProducts(productRepo)

	// Mock RabbitMQ client
	mockMQ = new(MockRabbitMQClient)
	mockMQ.On("PublishOrderCreated", mock.Anything).Return(nil)
	mockMQ.On("Close").Return(nil)

	// Initialize the app, injecting the mock MQ client
	app, authService, err = mainapp.NewApp()
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	code := m.Run()

	// Graceful Shutdown
	log.Println("Shutting down test environment...")
	if err := app.Shutdown(); err != nil {
		log.Printf("Error during Fiber shutdown: %v", err)
	}
	mockMQ.Close()

	os.Exit(code)
}

func TestServerStartupAndHealthCheck(t *testing.T) {
	// Get the configured port
	appPort := v.GetString("APP_PORT")
	if appPort == "" {
		appPort = ":8081" // Ensure tests use the correct port
	}

	// Start the server in a goroutine with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := app.Listen(appPort); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Test server listen error: %v", err)
			cancel() // Signal test failure if server cannot start
		}
		log.Printf("Test server stopped")
	}()
	defer cancel()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// --- Test Health Endpoint ---
	t.Run("HealthCheck", func(t *testing.T) {
		healthCheckURL := fmt.Sprintf("http://localhost%s/health", appPort)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthCheckURL, nil)
		if err != nil {
			t.Fatalf("Failed to create health check request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Health check request failed: %v", err)
		}
		defer func() {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status OK for health check; got %v", resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read health check response body: %v", err)
		}

		bodyString := string(bodyBytes)
		assert.Contains(t, bodyString, "\"status\":\"healthy\"", "Health check response body does not contain expected status")
	})

	// --- Test Unauthenticated Access ---
	t.Run("UnauthenticatedAccess", func(t *testing.T) {
		productsURL := fmt.Sprintf("http://localhost%s/api/v1/products", appPort)
		reqProducts, err := http.NewRequestWithContext(ctx, http.MethodGet, productsURL, nil)
		if err != nil {
			t.Fatalf("Failed to create products request: %v", err)
		}

		client := &http.Client{}
		respProducts, err := client.Do(reqProducts)
		if err != nil {
			t.Fatalf("Products request failed without token: %v", err)
		}
		defer func() {
			if respProducts != nil && respProducts.Body != nil {
				respProducts.Body.Close()
			}
		}()
		assert.Equal(t, http.StatusUnauthorized, respProducts.StatusCode, "Expected Unauthorized for /products without token")
	})

	// Graceful Shutdown is handled in TestMain now.
	// We need to ensure the test server is stopped cleanly before TestMain exits.
	// The defer cancel() in TestMain should handle this for the context.
	// We can add a small sleep to ensure shutdown logs are visible.
	time.Sleep(500 * time.Millisecond)
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
