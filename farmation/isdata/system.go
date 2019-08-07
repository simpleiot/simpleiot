package isdata

// Reboot is a message that causes the system to reboot
type Reboot struct{}

// ExportData is a message that causes the system to export faults
type ExportData struct{}

// ExportDataFinished is used to signal the system that export process has completed
type ExportDataFinished struct{}

// NoDiskPresent is used to trigger a dialog if no USB disk is inserted
type NoDiskPresent struct{}

// ErrWriteDisk is used to trigger a dialog if there was an error writing to a USB disk
type ErrWriteDisk struct{}
