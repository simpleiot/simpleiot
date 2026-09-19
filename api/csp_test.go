package api

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"testing"

	"github.com/simpleiot/simpleiot/frontend"
)

// TestImportMapHash keeps the hash the content security policy allows in
// step with the import map in index.html, since a browser refuses the
// script, and with it the whole UI, when they differ.
func TestImportMapHash(t *testing.T) {
	page, err := frontend.Content.ReadFile("public/index.html")
	if err != nil {
		t.Fatal(err)
	}

	m := regexp.MustCompile(`(?s)<script type="importmap">(.*?)</script>`).FindSubmatch(page)
	if m == nil {
		t.Fatal("index.html has no import map")
	}

	sum := sha256.Sum256(m[1])
	want := "sha256-" + base64.StdEncoding.EncodeToString(sum[:])
	if importMapHash != want {
		t.Fatalf("importMapHash is %v, index.html needs %v", importMapHash, want)
	}

	// no other inline script, since the policy would refuse it
	if n := len(regexp.MustCompile(`<script(?:\s[^>]*)?>`).FindAll(page, -1)); n != 3 {
		t.Fatalf("index.html has %v script tags; the policy allows the import map, elm.js, and main.js", n)
	}
}
