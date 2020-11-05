package data

import "time"

// User represents a user of the system
type User struct {
	ID        string `json:"id" boltholdKey:"ID"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Pass      string `json:"pass"`
}

// ToPoints converts a user structure into points
func (u *User) ToPoints() Points {
	now := time.Now()
	return Points{
		{Type: PointTypeFirstName, Time: now, Text: u.FirstName},
		{Type: PointTypeLastName, Time: now, Text: u.LastName},
		{Type: PointTypePhone, Time: now, Text: u.Phone},
		{Type: PointTypeEmail, Time: now, Text: u.Email},
		{Type: PointTypePass, Time: now, Text: u.Pass},
	}
}
