package services

import (
	"fmt"
	"log"
	"time"

	"toko/internal/models"
	"toko/internal/repositories"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles business logic for authentication and authorization.
type AuthService struct {
	userRepo   repositories.UserRepository
	jwtSecret  []byte
	tokenDurat time.Duration // Duration for which JWT is valid
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo repositories.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtSecret:  []byte(jwtSecret),
		tokenDurat: 24 * time.Hour, // Token valid for 24 hours
	}
}

// RegisterUser registers a new user, hashes their password, and saves them to the database.
func (s *AuthService) RegisterUser(user *models.User) error {
	// Check if username or email already exists
	if existingUser, err := s.userRepo.GetByUsername(user.Username); err == nil && existingUser != nil {
		return fmt.Errorf("username '%s' already taken", user.Username)
	}
	if existingUser, err := s.userRepo.GetByEmail(user.Email); err == nil && existingUser != nil {
		return fmt.Errorf("email '%s' already registered", user.Email)
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword) // Store the hashed password

	if err := s.userRepo.Create(user); err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}
	return nil
}

// LoginUser authenticates a user and returns a JWT token if successful.
func (s *AuthService) LoginUser(username, password string) (string, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		// It's good practice not to reveal if the username exists or not for security
		return "", fmt.Errorf("invalid credentials")
	}

	// Compare the provided password with the hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(s.tokenDurat).Unix(), // Token expiration time
		"iat":      time.Now().Unix(),                   // Issued at time
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses and validates a JWT token, returning the claims if valid.
func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the alg is what we expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		log.Printf("Token validation error: %v", err)
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
