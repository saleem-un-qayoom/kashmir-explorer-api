package tiles

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "11", "1450"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "11", "1450", "817.pbf"), []byte("pbf-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Get("/v1/tiles/contours/{z}/{x}/{y}.pbf", NewService(dir).Contour)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, dir
}

func TestContourServesTile(t *testing.T) {
	srv, _ := newTestServer(t)

	res, err := http.Get(srv.URL + "/v1/tiles/contours/11/1450/817.pbf")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/x-protobuf" {
		t.Errorf("Content-Type = %q, want application/x-protobuf", ct)
	}
	if cc := res.Header.Get("Cache-Control"); cc != "public, max-age=604800" {
		t.Errorf("Cache-Control = %q", cc)
	}
}

func TestContourMissingTile404s(t *testing.T) {
	srv, _ := newTestServer(t)

	res, err := http.Get(srv.URL + "/v1/tiles/contours/11/1450/9999.pbf")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}

func TestContourRejectsNonNumericCoords(t *testing.T) {
	srv, dir := newTestServer(t)

	// A file outside the tile tree that traversal must never reach.
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"/v1/tiles/contours/11/1450/..%2Fsecret.txt.pbf",
		"/v1/tiles/contours/a/b/c.pbf",
		"/v1/tiles/contours/-1/0/0.pbf",
	} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", path, res.StatusCode)
		}
	}
}
