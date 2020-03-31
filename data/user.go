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
	Roles     []Role    `json:"roles,omitempty"`
}

// A Role represents the role
// played by a user within an Org.
type Role struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"orgID"`
	OrgName     string    `json:"orgName"`
	Description string    `json:"description"`
}

// An Org represents a named collection of
// Users and Devices.
type Org struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Users   []User    `json:"users,omitempty"`
	Devices []Device  `json:"devices,omitempty"`
}
