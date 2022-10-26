package frontend

import (
	"fmt"
	"testing"
)

func TestEmbed(t *testing.T) {
	d, err := Content.ReadDir("output")
	if err != nil {
		t.Fatal("ReadDir returned: ", err)
	}
	for _, e := range d {
		fmt.Println("embed: ", e.Name())
	}
}
