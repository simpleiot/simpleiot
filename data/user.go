package data

import "time"

// User represents a user of the system
type User struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Pass      string `json:"pass"`
}

// ToPoints converts a user structure into points
func (u *User) ToPoints() Points {
	now := time.Now()
	pts := Points{
		NewPointString(PointTypeFirstName, "", u.FirstName),
		NewPointString(PointTypeLastName, "", u.LastName),
		NewPointString(PointTypePhone, "", u.Phone),
		NewPointString(PointTypeEmail, "", u.Email),
		NewPointString(PointTypePass, "", u.Pass),
	}
	for i := range pts {
		pts[i].Time = now
	}
	return pts
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
			ret.FirstName = p.Txt()
		case PointTypeLastName:
			ret.LastName = p.Txt()
		case PointTypeEmail:
			ret.Email = p.Txt()
		case PointTypePhone:
			ret.Phone = p.Txt()
		case PointTypePass:
			ret.Pass = p.Txt()
		}
	}

	return ret, nil
}
