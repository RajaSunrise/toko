package models

// User represents a user of the store.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"` // In a real app, this would be hashed.
}
