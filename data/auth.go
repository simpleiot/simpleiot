package data

// Auth is an authentication response.
type Auth struct {
	Token  string `json:"token"`
	IsRoot bool   `json:"isRoot"`
}
