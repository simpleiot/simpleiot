// +build tools

package tools

import (
	// genesis is used to generate static assets
	// to embed in binary
	_ "github.com/benbjohnson/genesis/cmd/genesis"
	_ "golang.org/x/lint/golint"
)
