package handlers

import (
	"fmt"
	"log"
	"strings"
	"toko/internal/models"
	"toko/internal/services"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// AuthHandler handles HTTP requests for authentication.
type AuthHandler struct {
	authService *services.AuthService
	validate    *validator.Validate
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validate:    validator.New(),
	}
}

// RegisterRoutes registers the authentication routes with the Fiber app.
func (h *AuthHandler) RegisterRoutes(router fiber.Router) {
	authRoutes := router.Group("/auth")
	authRoutes.Post("/register", h.HandleRegister)
	authRoutes.Post("/login", h.HandleLogin)
}

// HandleRegister handles new user registration.
func (h *AuthHandler) HandleRegister(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		log.Printf("Error parsing register request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	// Validate the user struct
	if err := h.validate.Struct(user); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		errorMessages := make(map[string]string)
		for _, e := range validationErrors {
			errorMessages[e.Field()] = fmt.Sprintf("Field '%s' failed on the '%s' tag", e.Field(), e.Tag())
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Validation failed",
			"errors":  errorMessages,
		})
	}

	if err := h.authService.RegisterUser(&user); err != nil {
		log.Printf("Error registering user: %v", err)
		if strings.Contains(err.Error(), "already taken") || strings.Contains(err.Error(), "already registered") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"message": "Registration failed",
				"error":   err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not register user",
			"error":   err.Error(),
		})
	}

	// For security, do not return the password hash
	user.Password = ""
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
		"user":    user,
	})
}

// LoginRequest represents the request body for login.
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// HandleLogin handles user login and issues a JWT token.
func (h *AuthHandler) HandleLogin(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing login request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	// Validate the login request
	if err := h.validate.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		errorMessages := make(map[string]string)
		for _, e := range validationErrors {
			errorMessages[e.Field()] = fmt.Sprintf("Field '%s' failed on the '%s' tag", e.Field(), e.Tag())
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Validation failed",
			"errors":  errorMessages,
		})
	}

	token, err := h.authService.LoginUser(req.Username, req.Password)
	if err != nil {
		log.Printf("Error during login for user %s: %v", req.Username, err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Authentication failed",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Login successful",
		"token":   token,
	})
}
