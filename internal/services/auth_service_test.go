package services_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"toko/internal/models"
	"toko/internal/services"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository is a mock implementation of repositories.UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(id string) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// TestMain is used to setup test environment
func TestMain(m *testing.M) {
	// Suppress logging during tests for cleaner output
	log.SetOutput(os.Stdout) // Changed to stdout to see logs if any, can be changed to ioutil.Discard
	code := m.Run()
	os.Exit(code)
}

func TestAuthService_RegisterUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	testJWTSecret := "test_jwt_secret"
	authService := services.NewAuthService(mockRepo, testJWTSecret)

	// Test successful registration
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	// Mock bcrypt hashing
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	user.Password = string(hashedPassword) // Set hashed password for mock verification if needed

	mockRepo.On("GetByUsername", user.Username).Return(nil, nil).Once()
	mockRepo.On("GetByEmail", user.Email).Return(nil, nil).Once()
	mockRepo.On("Create", mock.AnythingOfType("*models.User")).Return(nil).Once() // Use mock.AnythingOfType for any user object

	err := authService.RegisterUser(user)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Test username already taken
	mockRepo.On("GetByUsername", user.Username).Return(&models.User{ID: "1"}, nil).Once()
	err = authService.RegisterUser(user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username 'testuser' already taken")
	mockRepo.AssertExpectations(t)

	// Test email already registered
	mockRepo.On("GetByUsername", user.Username).Return(nil, nil).Once()
	mockRepo.On("GetByEmail", user.Email).Return(&models.User{ID: "1"}, nil).Once()
	err = authService.RegisterUser(user)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email 'test@example.com' already registered")
	mockRepo.AssertExpectations(t)
}

func TestAuthService_LoginUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	testJWTSecret := "test_jwt_secret"
	authService := services.NewAuthService(mockRepo, testJWTSecret)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &models.User{
		ID:       "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		Password: string(hashedPassword),
	}

	// Test successful login
	mockRepo.On("GetByUsername", user.Username).Return(user, nil).Once()
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123")) // Simulate the comparison
	assert.NoError(t, err)

	token, err := authService.LoginUser("testuser", "password123")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate the token structure (optional, but good to check)
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(testJWTSecret), nil
	})
	assert.NoError(t, err)
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, user.ID, claims["user_id"])
	assert.Equal(t, user.Username, claims["username"])
	mockRepo.AssertExpectations(t)

	// Test invalid credentials (wrong password)
	mockRepo.On("GetByUsername", user.Username).Return(user, nil).Once()
	_, err = authService.LoginUser("testuser", "wrongpassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
	mockRepo.AssertExpectations(t)

	// Test invalid credentials (user not found)
	mockRepo.On("GetByUsername", "nonexistentuser").Return(nil, fmt.Errorf("user with username nonexistentuser not found")).Once()
	_, err = authService.LoginUser("nonexistentuser", "password123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials") // Should return generic invalid credentials message
	mockRepo.AssertExpectations(t)
}

func TestAuthService_ValidateToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	testJWTSecret := "test_jwt_secret"
	authService := services.NewAuthService(mockRepo, testJWTSecret)

	// Generate a valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "user-123",
		"username": "testuser",
		"exp":      jwt.TimeFunc().Add(time.Hour).Unix(), // Expires in 1 hour
	})
	validTokenString, _ := token.SignedString([]byte(testJWTSecret))

	// Test valid token
	claims, err := authService.ValidateToken(validTokenString)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "user-123", claims["user_id"])
	assert.Equal(t, "testuser", claims["username"])

	// Test invalid token (wrong secret)
	invalidTokenString := "invalid.token.string"
	_, err = authService.ValidateToken(invalidTokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")

	// Test expired token
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "user-123",
		"username": "testuser",
		"exp":      jwt.TimeFunc().Add(-time.Hour).Unix(), // Expired 1 hour ago
	})
	expiredTokenString, _ := expiredToken.SignedString([]byte(testJWTSecret))
	_, err = authService.ValidateToken(expiredTokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}
