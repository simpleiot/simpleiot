//go:build tools
// +build tools

package tools

import (
	// The follow is used to prevent go mod tidy from removing
	// the entries from go.mod
	// genesis is used to generate static assets
	// to embed in binary
	_ "golang.org/x/lint/golint"
)
