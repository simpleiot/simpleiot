package isdata

// Reboot is a message that causes the system to reboot
type Reboot struct{}

// RestartApp is a message that fires a dialog warning user
// Used when timezone is changed in diagnostics/config
type RestartApp struct{}

// ExportData is a message that causes the system to export faults
type ExportData struct{}

// ExportDataFinished is used to signal the system that export process has completed
type ExportDataFinished struct{}

// ExportConfig is a message that causes the system to export config
type ExportConfig struct{}

// ExportConfigFinished is used to signal the system that export process has completed
type ExportConfigFinished struct{}

// ExportFieldProductTotals is a message that causes the system to export totals for
// each field and product
type ExportFieldProductTotals struct{}

// NoDiskPresent is used to trigger a dialog if no USB disk is inserted
type NoDiskPresent struct{}

// ErrWriteDisk is used to trigger a dialog if there was an error writing to a USB disk
type ErrWriteDisk struct{}

// ExportAlreadyInProcess triggers dialog
type ExportAlreadyInProcess struct{}

// NoNetworkConnection triggers a dialog to warn user of lost or nonexistent network
// connection when the IS is in Monitor and Notify mode, which relys on network for
// functionality
type NoNetworkConnection struct{}

// NoNetworkDialogDisplayed is message for the network thread that this dialog has been
// displayed. The network thread resets the timer for this dialog so that the user doesn't
// get it multiple times, like, if they switch to Notify mode, get the dialog, and then
// it fires again from the network thread
type NoNetworkDialogDisplayed struct{}

// DialogClose is a message to the app thread to close the dialog at Key in
// the State.Dialogs map
type DialogClose struct {
	Key string
}

// Shutdown indicates the system is shutting down
type Shutdown struct{}
