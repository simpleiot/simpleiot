package client

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateURL(t *testing.T) {
	for _, test := range []struct {
		base string
		ok   bool
	}{
		{"https://updates.example.com/dev", true},
		{"https://updates.example.com", true},
		{"http://updates.example.com/dev", false},
		{"ftp://updates.example.com/dev", false},
		{"updates.example.com/dev", false},
		{"https://", false},
		{"", false},
		{"://bad", false},
	} {
		u, err := updateURL(test.base, "files.txt")
		if test.ok && err != nil {
			t.Errorf("%q: unexpected error %v", test.base, err)
		}
		if !test.ok && err == nil {
			t.Errorf("%q: expected error, got %v", test.base, u)
		}
	}

	u, err := updateURL("https://updates.example.com/dev", "host_1.2.3.upd")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://updates.example.com/dev/host_1.2.3.upd" {
		t.Errorf("unexpected URL %v", u)
	}
}

// newUpdateServer serves files.txt and an update image over TLS. A
// request for missing.upd returns 404 and big.txt is over the list cap.
func newUpdateServer(t *testing.T) (*httptest.Server, updateFetcher) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/dev/files.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("host_1.0.0.upd\nhost_1.1.0.upd\n"))
	})
	mux.HandleFunc("/dev/host_1.1.0.upd", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("update image"))
	})
	mux.HandleFunc("/dev/empty.upd", func(_ http.ResponseWriter, _ *http.Request) {})
	mux.HandleFunc("/big/files.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", updateListMaxBytes+1)))
	})
	ts := httptest.NewTLSServer(mux)
	t.Cleanup(ts.Close)
	return ts, updateFetcher{list: ts.Client(), download: ts.Client()}
}

func TestUpdateFetchList(t *testing.T) {
	ts, f := newUpdateServer(t)

	lines, err := f.fetchList(ts.URL + "/dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 || lines[1] != "host_1.1.0.upd" {
		t.Errorf("unexpected list %q", lines)
	}

	if _, err := f.fetchList(ts.URL + "/missing"); err == nil {
		t.Error("expected error for 404")
	}

	if _, err := f.fetchList(ts.URL + "/big"); err == nil {
		t.Error("expected error for oversized files.txt")
	}

	if _, err := f.fetchList(strings.Replace(ts.URL, "https", "http", 1) + "/dev"); err == nil {
		t.Error("expected error for http URI")
	}
}

func TestUpdateFetchFile(t *testing.T) {
	ts, f := newUpdateServer(t)
	dir := t.TempDir()

	err := f.fetchFile(ts.URL+"/dev", "host_1.1.0.upd", dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "host_1.1.0.upd"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "update image" {
		t.Errorf("unexpected content %q", got)
	}

	// a 404 leaves no file behind
	if err := f.fetchFile(ts.URL+"/dev", "missing.upd", dir); err == nil {
		t.Error("expected error for 404")
	}
	if _, err := os.Stat(filepath.Join(dir, "missing.upd")); err == nil {
		t.Error("file left behind after failed download")
	}

	// neither does an empty download
	if err := f.fetchFile(ts.URL+"/dev", "empty.upd", dir); err == nil {
		t.Error("expected error for empty download")
	}
	if _, err := os.Stat(filepath.Join(dir, "empty.upd")); err == nil {
		t.Error("file left behind after empty download")
	}

	// a name with a directory component lands in dir
	if err := f.fetchFile(ts.URL, "dev/host_1.1.0.upd", dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "host_1.1.0.upd")); err != nil {
		t.Error(err)
	}
}
