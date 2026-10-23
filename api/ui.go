package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
)

// uiHandler serves the Svelte UI from its build output. The UI is
// a single-page app, so a path with no file is answered with index.html
// and the app routes it in the browser.
type uiHandler struct {
	fsys  fs.FS
	files http.Handler
	// dev rereads index.html for every request, so a rebuild on disk
	// does not leave the policy hashes stale
	dev bool
	// scriptSrc is the policy's script-src for the page, computed from
	// index.html when the handler is made
	scriptSrc string
}

func newUIHandler(fsys fs.FS, dev bool) *uiHandler {
	h := &uiHandler{fsys: fsys, files: http.FileServer(http.FS(fsys)), dev: dev}
	h.scriptSrc = h.computeScriptSrc()
	return h
}

var inlineScript = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

// uiScriptHashes returns the CSP hash of each inline script in a page.
// SvelteKit's fallback page starts the app with one.
func uiScriptHashes(page []byte) []string {
	var hashes []string
	for _, m := range inlineScript.FindAllSubmatch(page, -1) {
		sum := sha256.Sum256(m[1])
		hashes = append(hashes, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	return hashes
}

func (h *uiHandler) computeScriptSrc() string {
	page, err := fs.ReadFile(h.fsys, "index.html")
	if err != nil {
		return "'self'"
	}
	return strings.Join(append([]string{"'self'"}, uiScriptHashes(page)...), " ")
}

func (h *uiHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	scriptSrc := h.scriptSrc
	if h.dev {
		scriptSrc = h.computeScriptSrc()
	}
	res.Header().Set("Content-Security-Policy", contentSecurityPolicy(scriptSrc))

	p := path.Clean("/" + req.URL.Path)
	if p != "/" && p != "/index.html" {
		name := strings.TrimPrefix(p, "/")
		if st, err := fs.Stat(h.fsys, name); err == nil && !st.IsDir() {
			if strings.HasPrefix(p, "/_app/immutable/") {
				// hashed file names; a new build makes new names
				res.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			req.URL.Path = p
			h.files.ServeHTTP(res, req)
			return
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			http.Error(res, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if strings.HasPrefix(p, "/_app/") {
			http.NotFound(res, req)
			return
		}
	}

	page, err := fs.ReadFile(h.fsys, "index.html")
	if err != nil {
		http.NotFound(res, req)
		return
	}
	// the page names the current build's files, so it is checked each time
	res.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(res, req, "index.html", time.Time{}, bytes.NewReader(page))
}
