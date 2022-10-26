package frontend

import "embed"

// Content is a FS that holds the web UI assets
//go:embed public/*
var Content embed.FS
