package data

import "github.com/google/uuid"

// Role of user
type Role string

// define standard roles
const (
	RoleAdmin Role = "admin"
	RoleUser       = "user"
)

// UserRoles describes a users roles in an organization
type UserRoles struct {
	UserID uuid.UUID `json:"userId"`
	Roles  []Role    `json:"roles"`
}

// An Org represents a named collection of
// Users and Devices.
type Org struct {
	ID     uuid.UUID   `json:"id" boltholdKey:"ID"`
	Name   string      `json:"name"`
	Parent uuid.UUID   `json:"parent"`
	Users  []UserRoles `json:"users"`
}

// FindUsers returns users for specified role
func (o *Org) FindUsers(role Role) []uuid.UUID {
	var ret []uuid.UUID
	for _, ur := range o.Users {
		for _, r := range ur.Roles {
			if r == role {
				ret = append(ret, ur.UserID)
			}
		}
	}

	return ret
}
