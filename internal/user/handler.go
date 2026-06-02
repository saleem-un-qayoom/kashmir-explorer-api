package user

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

// Handler is the HTTP layer for the user domain. It owns request parsing and
// response writing only; all logic lives in Service.
type Handler struct {
	svc *Service
}

// New wires Repository → Service → Handler and returns the Handler, which is
// the package's public entry point (registered by router via AuthedRoutes).
func New(pool *pgxpool.Pool) *Handler {
	return &Handler{svc: NewService(NewRepository(pool))}
}

// Me godoc
// @Summary  Current user profile
// @Tags     user
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=user.Profile}
// @Failure  404 {object} response.Envelope
// @Router   /v1/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Profile(r.Context(), mw.UserID(r))
	if err != nil {
		response.FromError(w, r, err, "user not found")
		return
	}
	response.OK(w, p)
}

// Update godoc
// @Summary  Update current user profile (not yet implemented)
// @Tags     user
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope
// @Router   /v1/me [patch]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "update name, language, medical, insurance"})
}

// ListSaved godoc
// @Summary  List saved destinations
// @Tags     user
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]user.SavedDestination}
// @Router   /v1/saved [get]
func (h *Handler) ListSaved(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListSaved(r.Context(), mw.UserID(r))
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, items)
}

// Save godoc
// @Summary  Save a destination
// @Tags     user
// @Security BearerAuth
// @Produce  json
// @Param    destinationId path string true "Destination ID"
// @Success  200 {object} response.Envelope
// @Router   /v1/saved/{destinationId} [post]
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Save(r.Context(), mw.UserID(r), chi.URLParam(r, "destinationId")); err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, map[string]bool{"saved": true})
}

// Unsave godoc
// @Summary  Remove a saved destination
// @Tags     user
// @Security BearerAuth
// @Param    destinationId path string true "Destination ID"
// @Success  204
// @Router   /v1/saved/{destinationId} [delete]
func (h *Handler) Unsave(w http.ResponseWriter, r *http.Request) {
	// Errors are intentionally swallowed to preserve the prior idempotent
	// delete behavior (always 204, even if the row did not exist).
	_ = h.svc.Unsave(r.Context(), mw.UserID(r), chi.URLParam(r, "destinationId"))
	response.NoContent(w)
}

// ─── Itineraries ────────────────────────────────────────────

// ListItineraries godoc
// @Summary  List itineraries
// @Tags     user
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]user.Itinerary}
// @Router   /v1/itineraries [get]
func (h *Handler) ListItineraries(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListItineraries(r.Context(), mw.UserID(r))
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, items)
}

// CreateItinerary godoc
// @Summary  Create an itinerary (not yet implemented)
// @Tags     user
// @Security BearerAuth
// @Success  200 {object} response.Envelope
// @Router   /v1/itineraries [post]
func (h *Handler) CreateItinerary(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "create itinerary"})
}

// UpdateItinerary godoc
// @Summary  Update an itinerary (not yet implemented)
// @Tags     user
// @Security BearerAuth
// @Param    id path string true "Itinerary ID"
// @Success  200 {object} response.Envelope
// @Router   /v1/itineraries/{id} [put]
func (h *Handler) UpdateItinerary(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "update itinerary"})
}

// DeleteItinerary godoc
// @Summary  Delete an itinerary (not yet implemented)
// @Tags     user
// @Security BearerAuth
// @Param    id path string true "Itinerary ID"
// @Success  200 {object} response.Envelope
// @Router   /v1/itineraries/{id} [delete]
func (h *Handler) DeleteItinerary(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "delete itinerary"})
}
