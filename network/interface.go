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

// InterfaceConfig contains static information about
// an interface
type InterfaceConfig struct {
	Imei    string
	Sim     string
	Apn     string
	Version string
}

// Interface is an interface that network drivers implement
type Interface interface {
	Desc() string
	Configure() (InterfaceConfig, error)
	Connect() error
	GetStatus() (InterfaceStatus, error)
	Reset() error
	Enable(bool) error
}
