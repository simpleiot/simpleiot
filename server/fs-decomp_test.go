package server

import (
	"compress/gzip"
	"os"
	"testing"
)

func TestFsDecomp(t *testing.T) {
	f, err := os.Create("testfile.gz")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove("testfile.gz")

	w := gzip.NewWriter(f)

	testString := "Hi, this is a test"

	_, err = w.Write([]byte(testString))
	if err != nil {
		t.Fatal(err)
	}

	w.Close()
	f.Close()

	fs := os.DirFS(".")

	fsGz := newFsDecomp(fs)

	fd, err := fsGz.Open("testfile")
	if err != nil {
		t.Fatal(err)
	}

	fi, err := fd.Stat()
	if err != nil {
		t.Fatal(err)
	}

	if fi.Size() != int64(len(testString)) {
		t.Fatal("len is not correct")
	}

	if fi.Name() != "testfile" {
		t.Fatal("name is not correct: ", fi.Name())
	}

	buf := make([]byte, 50)
	c, err := fd.Read(buf)

	buf = buf[0:c]

	if string(buf) != testString {
		t.Fatal("Test string is not correct")
	}

}
