package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestFontsAreCacheable(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":   {Data: []byte("<html></html>")},
		"fonts/a.css":  {Data: []byte("@font-face{}")},
		"fonts/a.woff": {Data: []byte("x")},
	}
	app := &App{PublicHandler: http.FileServer(http.FS(fsys))}

	for path, want := range map[string]string{
		"/fonts/a.css":  "public, max-age=31536000, immutable",
		"/fonts/a.woff": "public, max-age=31536000, immutable",
		"/":             "",
	} {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%v: status %v", path, rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); got != want {
			t.Errorf("%v: Cache-Control %q, want %q", path, got, want)
		}
	}
}
