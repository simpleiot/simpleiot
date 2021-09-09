package frontend

import (
	"embed"
	"io/fs"
	"path"
)

//go:embed output/*
var content embed.FS

func Asset(name string) []byte {
	const filePath = "output"
	temp, err := content.ReadFile(path.Join(filePath, name))
	if err != nil {
		return nil
	}
	return temp
}

func FileSystem() fs.FS {
	return content
}
