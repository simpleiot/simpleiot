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
}

// A Role represents the role
// played by a user within an Org.
type Role struct {
	ID     uuid.UUID `json:"id"`
	OrgID  uuid.UUID `json:"orgID"`
	UserID uuid.UUID `json:"UserID"`
	Roles  []string  `json:"roles"`
}

// An Org represents a named collection of
// Users and Devices.
type Org struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// OrgDevice is used to bind devices to an org
type OrgDevice struct {
	ID       uuid.UUID `json:"id"`
	DeviceID string    `json:"deviceID"`
	OrgID    uuid.UUID `json:"orgID"`
}
