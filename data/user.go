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
		{Type: PointTypeNodeType, Time: now, Text: NodeTypeUser},
	}
}

// ToNode converts a user structure into a node
func (u *User) ToNode() Node {
	return Node{
		Type:   NodeTypeUser,
		Points: u.ToPoints(),
	}
}

// NodeToUser converts a node to a user
func NodeToUser(node Node) (User, error) {
	ret := User{}
	ret.ID = node.ID
	for _, p := range node.Points {
		switch p.Type {
		case PointTypeFirstName:
			ret.FirstName = p.Text
		case PointTypeLastName:
			ret.LastName = p.Text
		case PointTypeEmail:
			ret.Email = p.Text
		case PointTypePhone:
			ret.Phone = p.Text
		case PointTypePass:
			ret.Pass = p.Text
		}
	}

	return ret, nil
}
