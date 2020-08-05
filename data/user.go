package data

import (
	"github.com/google/uuid"
)

// User represents a user of the system
type User struct {
	ID        uuid.UUID `json:"id" boltholdKey:"ID"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Pass      string    `json:"pass"`
}
