package models

import "gorm.io/gorm"

// Product represents a product in the store.
type Product struct {
	ID          string  `json:"id" gorm:"primaryKey;type:varchar(36)" validate:"omitempty,uuid"`
	Name        string  `json:"name" validate:"required,min=3,max=100"`
	Description string  `json:"description" validate:"omitempty,max=500"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int     `json:"stock" validate:"gte=0"`
	gorm.Model          // Embed gorm.Model for CreatedAt, UpdatedAt, DeletedAt
}
