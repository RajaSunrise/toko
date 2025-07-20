package repositories

import (
	"fmt"
	"toko/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GORMUserRepository is a GORM implementation of UserRepository.
type GORMUserRepository struct {
	db *gorm.DB
}

// NewGORMUserRepository creates a new instance of GORMUserRepository.
func NewGORMUserRepository(db *gorm.DB) *GORMUserRepository {
	return &GORMUserRepository{
		db: db,
	}
}

// Create creates a new user in the database.
func (r *GORMUserRepository) Create(user *models.User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if err := r.db.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByUsername retrieves a user by their username from the database.
func (r *GORMUserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, "username = ?", username).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with username %s not found", username)
		}
		return nil, fmt.Errorf("failed to get user by username %s: %w", username, err)
	}
	return &user, nil
}

// GetByEmail retrieves a user by their email from the database.
func (r *GORMUserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, "email = ?", email).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with email %s not found", email)
		}
		return nil, fmt.Errorf("failed to get user by email %s: %w", email, err)
	}
	return &user, nil
}

// GetByID retrieves a user by their ID from the database.
func (r *GORMUserRepository) GetByID(id string) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get user by ID %s: %w", id, err)
	}
	return &user, nil
}
