package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionIsServedUncached(t *testing.T) {
	v1 := &V1{Version: "v1.2.3"}

	rec := httptest.NewRecorder()
	v1.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %v", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control %q, want no-cache", got)
	}
	var body struct{ Version string }
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Version != "v1.2.3" {
		t.Errorf("version %q, want v1.2.3", body.Version)
	}

	rec = httptest.NewRecorder()
	v1.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/version", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST status %v, want 405", rec.Code)
	}
}
