// Package data specifies data structures that are used on the wire.
package data

// Data provides all application data.
type Data struct {
	Orgs    []Org    `json:"orgs"`
	Users   []User   `json:"users"`
	Devices []Device `json:"devices"`
}
