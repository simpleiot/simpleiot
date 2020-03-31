package data

// Auth is an authentication response.
type Auth struct {
	Token     string `json:"token"`
	Privilege string `json:"privilege"`
}

// StandardResponse is the standard response to any request
type StandardResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	ID      string `json:"id,omitempty"`
}
