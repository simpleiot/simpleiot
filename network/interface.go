package network

// InterfaceStatus defines the status of an interface
type InterfaceStatus struct {
	Detected  bool
	Connected bool
	Operator  string
	Signal    int
	Rsrp      int
	Rsrq      int
	IP        string
}

// Interface is an interface that network drivers implement
type Interface interface {
	Desc() string
	Configure() error
	Connect() error
	GetStatus() (InterfaceStatus, error)
	Reset() error
}
