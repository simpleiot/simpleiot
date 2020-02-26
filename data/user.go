package data

import (
	"github.com/google/uuid"
)

// User represents a user of the system
type User struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	Pass      string    `json:"pass"`
	Admin     bool      `json:"admin"`
}
