package network

// DummyInterface is an interface that always reports detected/connected
type DummyInterface struct {
}

// NewDummyInterface constructor
func NewDummyInterface() *DummyInterface {
	return &DummyInterface{}
}

// Desc returns description
func (d *DummyInterface) Desc() string {
	return "net"
}

// Configure stub
func (d *DummyInterface) Configure() (InterfaceConfig, error) {
	return InterfaceConfig{}, nil
}

// Connect stub
func (d *DummyInterface) Connect() error {
	return nil
}

// GetStatus return interface status
func (d *DummyInterface) GetStatus() (InterfaceStatus, error) {
	return InterfaceStatus{
		Detected:  true,
		Connected: true,
	}, nil
}

// Reset stub
func (d *DummyInterface) Reset() error {
	return nil
}

// Enable or disable interface
func (d *DummyInterface) Enable(_ bool) error {
	return nil
}
