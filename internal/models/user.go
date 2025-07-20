package models

import "gorm.io/gorm"

// User represents a user of the store.
type User struct {
	ID         string `json:"id" gorm:"primaryKey;type:varchar(36)" validate:"omitempty,uuid"`
	Username   string `json:"username" gorm:"uniqueIndex;type:varchar(100)" validate:"required,min=3,max=100"`
	Email      string `json:"email" gorm:"uniqueIndex;type:varchar(255)" validate:"required,email"`
	Password   string `gorm:"type:varchar(255)" validate:"required,min=6"` // No json tag for security
	gorm.Model        // Embed gorm.Model for CreatedAt, UpdatedAt, DeletedAt
}
