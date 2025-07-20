package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"toko/internal/handlers"
	"toko/internal/middleware"
	"toko/internal/models"
	"toko/internal/repositories"
	"toko/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupApp sets up a Fiber app for testing with in-memory SQLite and all handlers/services.
func setupApp() (*fiber.App, *services.AuthService, error) {
	// Configure Viper for testing
	viper.SetDefault("JWT_SECRET", "test_jwt_secret")
	viper.AutomaticEnv()
	jwtSecret := viper.GetString("JWT_SECRET")

	// Initialize in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to in-memory database: %w", err)
	}

	// Auto-migrate models
	err = db.AutoMigrate(&models.Product{}, &models.User{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	// Initialize Repositories
	productRepo := repositories.NewGORMProductRepository(db)
	userRepo := repositories.NewGORMUserRepository(db)
	orderRepo := repositories.NewMockOrderRepository() // Using mock for order for simplicity in this test

	// Initialize Services
	productService := services.NewProductService(productRepo)
	orderService := services.NewOrderService(orderRepo, productRepo, nil) // nil for RabbitMQ client
	authService := services.NewAuthService(userRepo, jwtSecret)

	// Initialize Handlers
	productHandler := handlers.NewProductHandler(productService)
	orderHandler := handlers.NewOrderHandler(orderService)
	authHandler := handlers.NewAuthHandler(authService)

	app := fiber.New()

	// API Routes
	apiV1 := app.Group("/api/v1")

	// Authentication routes (public)
	authHandler.RegisterRoutes(apiV1)

	// Protected routes (require JWT authentication)
	protectedRoutes := apiV1.Group("", middleware.AuthRequired(authService))

	// Register product routes
	productHandler.RegisterRoutes(protectedRoutes)
	// Register order routes
	orderHandler.RegisterRoutes(protectedRoutes)

	// Seed some initial products (optional, but good for testing GET all)
	seedProductsForTest(productRepo)

	return app, authService, nil
}

// seedProductsForTest populates the product repository for tests.
func seedProductsForTest(repo repositories.ProductRepository) {
	products := []models.Product{
		{Name: "Test Laptop", Description: "For testing purposes", Price: 1000.00, Stock: 5},
		{Name: "Test Monitor", Description: "Another test item", Price: 200.00, Stock: 10},
	}
	for i := range products {
		if err := repo.Create(&products[i]); err != nil {
			log.Printf("Failed to seed product %s: %v", products[i].Name, err)
		}
	}
}

// TestMain runs setup and teardown for all tests
func TestMain(m *testing.M) {
	// Suppress logging during tests for cleaner output
	log.SetOutput(ioutil.Discard)
	// Run tests
	code := m.Run()
	// You could add global teardown here if needed
	os.Exit(code)
}

func TestAuthRegisterAndLogin(t *testing.T) {
	app, authService, err := setupApp()
	assert.NoError(t, err)

	// Test Registration
	userToRegister := map[string]string{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(userToRegister)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1) // -1 for no timeout
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var registerResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registerResp)
	assert.NoError(t, err)
	assert.Equal(t, "User registered successfully", registerResp["message"])
	resp.Body.Close()

	// Test Duplicate Registration (username)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()

	// Test Login
	loginCredentials := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	jsonBody, _ = json.Marshal(loginCredentials)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]string
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	assert.NoError(t, err)
	assert.Contains(t, loginResp, "token")
	assert.NotEmpty(t, loginResp["token"])
	resp.Body.Close()

	// Optionally, validate the token with authService
	claims, err := authService.ValidateToken(loginResp["token"])
	assert.NoError(t, err)
	assert.Equal(t, "testuser", claims["username"])
	assert.Contains(t, claims, "user_id")
}

func TestToProductEndpointsWithoutAuth(t *testing.T) {
	app, _, err := setupApp()
	assert.NoError(t, err)

	// First, register and log in a user to get a token
	userToRegister := map[string]string{
		"username": "authuser",
		"email":    "auth@example.com",
		"password": "securepassword",
	}
	jsonBody, _ := json.Marshal(userToRegister)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	loginCredentials := map[string]string{
		"username": "authuser",
		"password": "securepassword",
	}
	jsonBody, _ = json.Marshal(loginCredentials)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var loginResp map[string]string
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	assert.NoError(t, err)
	token := loginResp["token"]
	assert.NotEmpty(t, token)
	resp.Body.Close()

	// --- Test GET /products (protected) ---
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var products []models.Product
	err = json.NewDecoder(resp.Body).Decode(&products)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(products), 2) // Should contain seeded products
	resp.Body.Close()

	// --- Test POST /products (protected) ---
	newProduct := map[string]interface{}{
		"name":        "Smartphone",
		"description": "Latest model smartphone",
		"price":       799.99,
		"stock":       50,
	}
	jsonBody, _ = json.Marshal(newProduct)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var createdProduct models.Product
	err = json.NewDecoder(resp.Body).Decode(&createdProduct)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdProduct.ID)
	assert.Equal(t, newProduct["name"], createdProduct.Name)
	resp.Body.Close()

	// --- Test GET /products/:id (protected) ---
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products/"+createdProduct.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var fetchedProduct models.Product
	err = json.NewDecoder(resp.Body).Decode(&fetchedProduct)
	assert.NoError(t, err)
	assert.Equal(t, createdProduct.ID, fetchedProduct.ID)
	resp.Body.Close()

	// --- Test PUT /products/:id (protected) ---
	updatedProductData := map[string]interface{}{
		"name":        "Smartphone Pro",
		"description": "Latest model smartphone pro edition",
		"price":       899.99,
		"stock":       45,
	}
	jsonBody, _ = json.Marshal(updatedProductData)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/products/"+createdProduct.ID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var updatedProduct models.Product
	err = json.NewDecoder(resp.Body).Decode(&updatedProduct)
	assert.NoError(t, err)
	assert.Equal(t, createdProduct.ID, updatedProduct.ID)
	assert.Equal(t, updatedProductData["name"], updatedProduct.Name)
	resp.Body.Close()

	// --- Test DELETE /products/:id (protected) ---
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+createdProduct.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var deleteResp map[string]string
	err = json.NewDecoder(resp.Body).Decode(&deleteResp)
	assert.NoError(t, err)
	assert.Contains(t, deleteResp["message"], "deleted successfully")
	resp.Body.Close()

	// Verify deletion
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products/"+createdProduct.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestProductEndpointsWithoutAuth(t *testing.T) {
	app, _, err := setupApp()
	assert.NoError(t, err)

	// Test GET /products without token
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	// Test POST /products without token
	newProduct := map[string]interface{}{
		"name":  "Unauthorized Product",
		"price": 100.0,
		"stock": 10,
	}
	jsonBody, _ := json.Marshal(newProduct)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}
