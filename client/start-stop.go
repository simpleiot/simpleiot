package client

// RunStop is an interface that implements the Run() and Stop() methods.
// This pattern is used wherever long running processes are required.
// Warning!!! Stop() may get called after Run() has exited when using
// mechanisms like run.Group, so be sure that Stop() never blocks -- it must
// return for things to work properly.
type RunStop interface {
	Run() error
	Stop(error)
}
