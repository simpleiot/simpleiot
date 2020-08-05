package data

// Condition defines parameters to look for in a sample. Either SampleType or SampleID
// (or both) can be set. They can't both be "".
type Condition struct {
	SampleType string  `json:"sampleType"`
	SampleID   string  `json:"sampleID"`
	Value      float64 `json:"value"`
}

// Rule defines a conditions and actions that are run if condition is true. Global indicates
// the rule applies to all Devices.
type Rule struct {
	ID          string      `json:"id" boltholdKey:"ID"`
	Global      bool        `json:"global"`
	Description string      `json:"description"`
	Conditions  []Condition `json:"conditions"`
}
