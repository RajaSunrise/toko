package repositories

import "toko/internal/models"

// UserRepository defines the interface for user data access.
type UserRepository interface {
	Create(user *models.User) error
	GetByUsername(username string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByID(id string) (*models.User, error)
}
