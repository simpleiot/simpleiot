package frontend

import (
	"embed"
	"net/http"
	"path"
)

//go:embed output/*
var content embed.FS

func Asset(name string) []byte {
	const filePath = "frontend/output"
	temp, err := content.ReadFile(path.Join(filePath, name))
	if err != nil {
		return nil
	}
	return temp
}

func FileSystem() http.FileSystem {
	return http.FS(content)
}
