package data

import "errors"

// DefaultUserEmail and DefaultUserPass are the credentials of the account a
// new instance starts with.
const (
	DefaultUserEmail = "admin"
	DefaultUserPass  = "admin"
)

// ErrDefaultPassword is the reply to a sign-in with the default password on
// an instance that has an auth token: setting a token says the instance is
// deployed, and a deployed instance does not accept the password every
// instance starts with.
var ErrDefaultPassword = errors.New(
	"the default password is not accepted when an auth token is set; set a new password for this account")

// Auth is an authentication response.
type Auth struct {
	Token string `json:"token"`
	Email string `json:"email"`
	// DefaultPassword is set when the account signed in with the password
	// every instance starts with, so the UI can ask for it to be changed.
	DefaultPassword bool `json:"defaultPassword,omitempty"`
	// NoAuthToken is set when the instance has no auth token, so the UI
	// can say that it accepts connections without credentials.
	NoAuthToken bool `json:"noAuthToken,omitempty"`
}
