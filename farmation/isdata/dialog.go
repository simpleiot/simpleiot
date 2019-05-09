package isdata

// DialogMsg is used to display a dialog box and optionally get return value
// via a callback
type DialogMsg struct {
	Text     string
	Ok       bool
	Yes      bool
	No       bool
	Callback func(yes bool)
}
