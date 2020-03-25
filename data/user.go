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
	Roles     []Role    `json:"roles"`
}

type Role struct {
	OrgID       uuid.UUID `json:"orgID"`
	OrgName     string    `json:"orgName"`
	Description string    `json:"description"`
}
