package frontend

import (
	"fmt"
	"io/fs"
	"testing"
)

func TestEmbed(t *testing.T) {
	fmt.Println("Root dir -----------------")
	d, err := Content.ReadDir(".")
	if err != nil {
		t.Fatal("ReadDir returned: ", err)
	}
	for _, e := range d {
		fmt.Println("embed: ", e.Name())
	}

	fmt.Println("public dir -----------------")
	d, err = Content.ReadDir("public")
	if err != nil {
		t.Fatal("ReadDir returned: ", err)
	}
	for _, e := range d {
		fmt.Println("embed: ", e.Name())
	}

	fmt.Println("subtree public walk -----------------")
	st, err := fs.Sub(Content, "public")
	if err != nil {
		t.Fatal("Error subtree: ", err)
	}

	err = fs.WalkDir(st, ".", func(path string, _ fs.DirEntry, _ error) error {
		fmt.Println("embed: ", path)
		return nil
	})

	if err != nil {
		t.Fatal("Walkdir error: ", err)
	}

	fmt.Println("subtree public readir -----------------")
	d, err = fs.ReadDir(st, ".")
	if err != nil {
		t.Fatal("ReadDir returned: ", err)
	}
	for _, e := range d {
		fmt.Println("embed: ", e.Name())
	}
}
