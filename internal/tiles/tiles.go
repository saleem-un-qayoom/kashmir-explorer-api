// Package tiles serves the pre-built contour vector-tile tree used by the
// mobile outdoor map style ({z}/{x}/{y}.pbf, layer "contour", field "ele").
//
// The tileset is generated from free SRTM elevation data by
// scripts/build-contour-tiles.sh — at Docker build time in production, or
// via `make contour-tiles` for local dev. Tiles are static files on disk;
// a missing tile (outside the built bbox, or tileset not generated yet)
// returns 404, which MapLibre treats as "no data here", so the app
// degrades gracefully to a contour-less map.
package tiles

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Service struct {
	dir string
}

// NewService serves tiles from dir (CONTOUR_TILES_DIR).
func NewService(dir string) *Service {
	return &Service{dir: dir}
}

// Contour handles GET /v1/tiles/contours/{z}/{x}/{y}.pbf.
func (s *Service) Contour(w http.ResponseWriter, r *http.Request) {
	// Parsing as integers doubles as path-traversal protection — anything
	// that isn't a plain tile coordinate 404s before touching the disk.
	z, errZ := strconv.Atoi(chi.URLParam(r, "z"))
	x, errX := strconv.Atoi(chi.URLParam(r, "x"))
	y, errY := strconv.Atoi(chi.URLParam(r, "y"))
	if errZ != nil || errX != nil || errY != nil || z < 0 || z > 22 || x < 0 || y < 0 {
		http.NotFound(w, r)
		return
	}

	f, err := os.Open(filepath.Join(s.dir, strconv.Itoa(z), strconv.Itoa(x), strconv.Itoa(y)+".pbf"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Contours only change when the tileset is rebuilt and redeployed —
	// let clients and MapLibre's tile cache hold them for a week.
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeContent(w, r, st.Name(), st.ModTime(), f)
}
