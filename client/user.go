package client

// User represents a user node
type User struct {
	ID        string `node:"id"`
	Parent    string `node:"parent"`
	FirstName string `point:"firstName"`
	LastName  string `point:"lastName"`
	Phone     string `point:"phone"`
	Email     string `point:"email"`
	Pass      string `point:"pass"`
}
