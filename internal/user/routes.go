package user

import "github.com/go-chi/chi/v5"

// AuthedRoutes registers the user routes that require an authenticated session.
// It is mounted by internal/router inside the /v1 group that applies mw.Auth.
func (h *Handler) AuthedRoutes(r chi.Router) {
	r.Get("/me", h.Me)
	r.Patch("/me", h.Update)

	r.Route("/saved", func(r chi.Router) {
		r.Get("/", h.ListSaved)
		r.Post("/{destinationId}", h.Save)
		r.Delete("/{destinationId}", h.Unsave)
	})

	r.Route("/itineraries", func(r chi.Router) {
		r.Get("/", h.ListItineraries)
		r.Post("/", h.CreateItinerary)
		r.Put("/{id}", h.UpdateItinerary)
		r.Delete("/{id}", h.DeleteItinerary)
	})
}
