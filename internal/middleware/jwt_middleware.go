package middleware

import (
	"log"
	"strings"

	"toko/internal/services"

	"github.com/gofiber/fiber/v2"
)

// AuthRequired is a Fiber middleware to check for a valid JWT token.
func AuthRequired(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Authorization header is required",
			})
		}

		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Authorization header format must be 'Bearer <token>'",
			})
		}

		tokenString := parts[1]

		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			log.Printf("JWT validation failed: %v", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid or expired token",
				"error":   err.Error(),
			})
		}

		// Store claims in Fiber context for subsequent handlers
		c.Locals("user_id", claims["user_id"])
		c.Locals("username", claims["username"])

		// Continue to the next handler
		return c.Next()
	}
}
