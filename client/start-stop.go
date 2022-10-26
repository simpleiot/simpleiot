package client

// StartStop is an interface that implements the Start() and Stop() methods.
// This pattern is used wherever long running processes are required.
type StartStop interface {
	Start() error
	Stop(error)
}
