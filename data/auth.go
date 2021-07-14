package data

// Auth is an authentication response.
type Auth struct {
	Token string `json:"token"`
	Email string `json:"email"`
}
