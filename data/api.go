package data

// Response is the standard response to any request
type Response struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
