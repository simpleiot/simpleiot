package api

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/simpleiot/simpleiot/ui"
)

// TestUIScriptHashes checks the policy the /v2 page is served with allows
// the inline script that starts the app, since a browser refuses the
// script, and with it the whole UI, when the hash does not match.
func TestUIScriptHashes(t *testing.T) {
	build, err := fs.Sub(ui.Content, "build")
	if err != nil {
		t.Fatal(err)
	}
	page, err := fs.ReadFile(build, "index.html")
	if err != nil {
		t.Fatal(err)
	}

	// SvelteKit's fallback page has one inline script; any other would
	// also have to be allowed
	if n := len(regexp.MustCompile(`<script(?:\s[^>]*)?>`).FindAll(page, -1)); n != 1 {
		t.Fatalf("index.html has %v script tags, want the one that starts the app", n)
	}
	hashes := uiScriptHashes(page)
	if len(hashes) != 1 {
		t.Fatalf("found %v inline scripts, want 1", len(hashes))
	}

	rec := httptest.NewRecorder()
	newUIHandler(build, false).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self' "+hashes[0]+";") {
		t.Fatalf("policy %q does not allow %v", csp, hashes[0])
	}
}

func TestUIFallback(t *testing.T) {
	page := "<html><script>start()</script></html>"
	fsys := fstest.MapFS{
		"index.html":                {Data: []byte(page)},
		"siot-logo.svg":             {Data: []byte("<svg/>")},
		"_app/version.json":         {Data: []byte("{}")},
		"_app/immutable/entry/a.js": {Data: []byte("a()")},
	}
	app := &App{UIHandler: newUIHandler(fsys, false)}

	for _, c := range []struct {
		path   string
		status int
		body   string
		cache  string
	}{
		{"/v2", http.StatusOK, page, "no-cache"},
		{"/v2/", http.StatusOK, page, "no-cache"},
		{"/v2/sign-in", http.StatusOK, page, "no-cache"},
		{"/v2/nodes/abc-123", http.StatusOK, page, "no-cache"},
		{"/v2/siot-logo.svg", http.StatusOK, "<svg/>", ""},
		{"/v2/_app/version.json", http.StatusOK, "{}", ""},
		{"/v2/_app/immutable/entry/a.js", http.StatusOK, "a()", "public, max-age=31536000, immutable"},
		{"/v2/_app/immutable/entry/missing.js", http.StatusNotFound, "", ""},
	} {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		if rec.Code != c.status {
			t.Errorf("%v: status %v, want %v", c.path, rec.Code, c.status)
			continue
		}
		if c.body != "" && rec.Body.String() != c.body {
			t.Errorf("%v: body %q, want %q", c.path, rec.Body.String(), c.body)
		}
		if got := rec.Header().Get("Cache-Control"); got != c.cache {
			t.Errorf("%v: Cache-Control %q, want %q", c.path, got, c.cache)
		}
		if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, uiScriptHashes([]byte(page))[0]) {
			t.Errorf("%v: policy %q does not allow the page script", c.path, csp)
		}
	}

	// the Elm UI keeps its own policy
	rec := httptest.NewRecorder()
	(&App{PublicHandler: http.NotFoundHandler(), UIHandler: app.UIHandler}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, importMapHash) {
		t.Errorf("policy at / %q lacks the import map hash", csp)
	}
}
