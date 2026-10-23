// Package ui embeds the built Svelte UI served at /v2.
package ui

import "embed"

// Content holds the build output. The all: prefix is needed because
// SvelteKit writes its assets to build/_app, and go:embed otherwise skips
// names that start with an underscore.
//
//go:embed all:build
var Content embed.FS
