package isdata

// Reboot is a message that causes the system to reboot
type Reboot struct{}

// ExportFaults is a message that causes the system to export faults
type ExportFaults struct{}

// ExportFaultsFinished is used to signal the system that export process has completed
type ExportFaultsFinished struct{}
