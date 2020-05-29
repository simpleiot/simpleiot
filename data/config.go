package data

// would like to create a type that can represent most config items and then
// a device (or even IO) configuration can just be an array of config entities
// Types of data in a config:
//  - string
//  - float
//  - bool

// Config defines parameter of config
type Config struct {
	NAME  string `json:"id"`
	IO    string
	Value float64 `json:"value"`
}
