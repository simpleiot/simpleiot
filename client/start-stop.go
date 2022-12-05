package client

// StartStop is an interface that implements the Start() and Stop() methods.
// This pattern is used wherever long running processes are required.
// Warning!!! Stop() may get called after Start() has exitted when using
// mechanisms like run.Group, so be sure that Stop() never blocks -- it must
// return for things to work properly.
type StartStop interface {
	Start() error
	Stop(error)
}
