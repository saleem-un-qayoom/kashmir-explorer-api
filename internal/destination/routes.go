package destination

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// PublicRoutes registers the unauthenticated destination + taxonomy routes off
// the /v1 root it is handed.
//
// photoSpots is injected because the legacy route
// GET /v1/destinations/{slug}/photo-spots lives inside the /destinations group
// but is served by the photo package; passing the handler keeps the whole
// /destinations subtree owned here without importing photo.
func (h *Handler) PublicRoutes(r chi.Router, photoSpots http.HandlerFunc) {
	r.Route("/destinations", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/featured", h.Featured)
		r.Get("/trending", h.Trending)
		r.Get("/nearby", h.Nearby)
		r.Get("/map", h.Bbox)
		r.Get("/{slug}", h.Get)
		r.Get("/{slug}/photo-spots", photoSpots)
	})

	// Taxonomy reads are mounted at the /v1 root (not under /destinations).
	// The {id} reads intentionally reuse the admin getters, matching the
	// pre-refactor public routing.
	r.Get("/categories", h.Categories)
	r.Get("/categories/{id}", h.AdminCategoryGet)
	r.Get("/regions", h.Regions)
	r.Get("/regions/{id}", h.AdminRegionGet)
}

// AdminRoutes registers the admin destination + taxonomy CRUD off the
// /v1/admin root (which already applies mw.Auth + mw.RequireAdmin).
func (h *Handler) AdminRoutes(r chi.Router) {
	// Destinations
	r.Get("/destinations", h.AdminList)
	r.Get("/destinations/{id}", h.AdminGet)
	r.Post("/destinations", h.AdminCreate)
	r.Put("/destinations/{id}", h.AdminUpdate)
	r.Delete("/destinations/{id}", h.AdminDelete)
	r.Post("/destinations/{id}/restore", h.AdminRestore)
	r.Delete("/destinations/{id}/permanent", h.AdminDeletePermanent)

	// Categories
	r.Get("/categories/{id}", h.AdminCategoryGet)
	r.Post("/categories", h.AdminCategoryCreate)
	r.Put("/categories/{id}", h.AdminCategoryUpdate)
	r.Delete("/categories/{id}", h.AdminCategoryDelete)

	// Regions
	r.Get("/regions/{id}", h.AdminRegionGet)
	r.Post("/regions", h.AdminRegionCreate)
	r.Put("/regions/{id}", h.AdminRegionUpdate)
	r.Delete("/regions/{id}", h.AdminRegionDelete)
}
