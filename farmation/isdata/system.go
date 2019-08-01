package isdata

// Reboot is a message that causes the system to reboot
type Reboot struct{}

// ExportData is a message that causes the system to export faults
type ExportData struct{}

// ExportDataFinished is used to signal the system that export process has completed
type ExportDataFinished struct{}
