package destination

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

// Handler is the HTTP layer for the destination domain.
type Handler struct {
	svc *Service
}

// New wires Repository → Service → Handler and returns the Handler, the
// package's public entry point (registered by router via PublicRoutes/AdminRoutes).
func New(pool *pgxpool.Pool) *Handler {
	return &Handler{svc: NewService(NewRepository(pool))}
}

// ─── Public ─────────────────────────────────────────────────────

// List godoc
// @Summary  List destinations
// @Tags     destinations
// @Produce  json
// @Param    region   query string false "Region slug"
// @Param    category query string false "Category slug"
// @Param    limit    query int    false "Page size (default 24, max 100)"
// @Param    offset   query int    false "Offset"
// @Success  200 {object} response.Envelope{data=[]destination.Destination}
// @Router   /v1/destinations [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	region := q.Get("region")
	category := q.Get("category")
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	out, err := h.svc.List(r.Context(), region, category, limit, offset)
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, out)
}

// Featured godoc
// @Summary  Featured destinations
// @Tags     destinations
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]destination.FeaturedDestination}
// @Router   /v1/destinations/featured [get]
func (h *Handler) Featured(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Featured(r.Context())
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, out)
}

// Trending godoc
// @Summary  Trending destinations
// @Tags     destinations
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]destination.TrendingDestination}
// @Router   /v1/destinations/trending [get]
func (h *Handler) Trending(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Trending(r.Context())
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, out)
}

// Nearby godoc
// @Summary  Destinations near a point
// @Tags     destinations
// @Produce  json
// @Param    lat       query number true  "Latitude"
// @Param    lng       query number true  "Longitude"
// @Param    radius_km query number false "Radius in km (default 20)"
// @Param    limit     query int    false "Limit (default 10)"
// @Success  200 {object} response.Envelope{data=[]destination.NearbyDestination}
// @Router   /v1/destinations/nearby [get]
func (h *Handler) Nearby(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lat, _ := strconv.ParseFloat(q.Get("lat"), 64)
	lng, _ := strconv.ParseFloat(q.Get("lng"), 64)
	radius, _ := strconv.ParseFloat(q.Get("radius_km"), 64)
	if radius == 0 {
		radius = 20
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 10
	}

	out, err := h.svc.Nearby(r.Context(), lng, lat, radius, limit)
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, out)
}

// Bbox godoc
// @Summary  Destination pins within a bounding box
// @Tags     destinations
// @Produce  json
// @Param    bbox query string true "minLat,minLng,maxLat,maxLng"
// @Success  200 {object} response.Envelope{data=[]destination.MapPin}
// @Failure  400 {object} response.Envelope
// @Router   /v1/destinations/map [get]
func (h *Handler) Bbox(w http.ResponseWriter, r *http.Request) {
	parts := splitFloats(r.URL.Query().Get("bbox"))
	if len(parts) != 4 {
		response.BadRequest(w, "bbox must be minLat,minLng,maxLat,maxLng")
		return
	}
	minLat, minLng, maxLat, maxLng := parts[0], parts[1], parts[2], parts[3]
	out, err := h.svc.Bbox(r.Context(), minLng, minLat, maxLng, maxLat)
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, out)
}

// Get godoc
// @Summary  Destination detail by slug
// @Tags     destinations
// @Produce  json
// @Param    slug path string true "Destination slug"
// @Success  200 {object} response.Envelope{data=destination.Destination}
// @Failure  404 {object} response.Envelope
// @Router   /v1/destinations/{slug} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.GetBySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		response.FromError(w, r, err, "destination not found")
		return
	}
	response.OK(w, d)
}

// Categories godoc
// @Summary  List categories
// @Tags     taxonomy
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]destination.Category}
// @Router   /v1/categories [get]
func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Categories(r.Context())
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, out)
}

// Regions godoc
// @Summary  List regions
// @Tags     taxonomy
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]destination.Region}
// @Router   /v1/regions [get]
func (h *Handler) Regions(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Regions(r.Context())
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, out)
}

// ─── Admin: destinations ────────────────────────────────────────

// AdminList godoc
// @Summary  List destinations (admin, includes unpublished/deleted)
// @Tags     admin-destinations
// @Security BearerAuth
// @Produce  json
// @Param    status query string false "published|unpublished|deleted"
// @Success  200 {object} response.Envelope{data=[]destination.AdminDest}
// @Router   /v1/admin/destinations [get]
func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.AdminList(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, out)
}

// AdminGet godoc
// @Summary  Destination detail (admin)
// @Tags     admin-destinations
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Destination ID"
// @Success  200 {object} response.Envelope{data=destination.AdminDest}
// @Router   /v1/admin/destinations/{id} [get]
func (h *Handler) AdminGet(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.AdminGet(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.FromError(w, r, err, "destination not found")
		return
	}
	response.OK(w, d)
}

// AdminCreate godoc
// @Summary  Create a destination (admin)
// @Tags     admin-destinations
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body destination.AdminDestInput true "Destination"
// @Success  200 {object} response.Envelope
// @Failure  400 {object} response.Envelope
// @Router   /v1/admin/destinations [post]
func (h *Handler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var in AdminDestInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if in.Name == "" || in.Slug == "" {
		response.BadRequest(w, "name and slug required")
		return
	}
	id, err := h.svc.AdminCreate(r.Context(), in)
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, map[string]string{"id": id})
}

// AdminUpdate godoc
// @Summary  Update a destination (admin)
// @Tags     admin-destinations
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id   path string              true "Destination ID"
// @Param    body body destination.AdminDestInput true "Destination"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/destinations/{id} [put]
func (h *Handler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in AdminDestInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.AdminUpdate(r.Context(), id, in); err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

// AdminDelete godoc
// @Summary  Soft-delete a destination (admin)
// @Tags     admin-destinations
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Destination ID"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/destinations/{id} [delete]
func (h *Handler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.AdminSoftDelete(r.Context(), id); err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, map[string]string{"deleted": id})
}

// AdminRestore godoc
// @Summary  Restore a soft-deleted destination (admin)
// @Tags     admin-destinations
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Destination ID"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/destinations/{id}/restore [post]
func (h *Handler) AdminRestore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.AdminRestore(r.Context(), id); err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, map[string]string{"restored": id})
}

// AdminDeletePermanent godoc
// @Summary  Permanently delete a destination (admin)
// @Tags     admin-destinations
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Destination ID"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/destinations/{id}/permanent [delete]
func (h *Handler) AdminDeletePermanent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.AdminDeletePermanent(r.Context(), id); err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, map[string]string{"deleted": id})
}

// ─── Admin: categories ──────────────────────────────────────────

// AdminCategoryGet godoc
// @Summary  Category detail (admin)
// @Tags     taxonomy
// @Produce  json
// @Param    id path string true "Category ID"
// @Success  200 {object} response.Envelope{data=destination.Category}
// @Router   /v1/categories/{id} [get]
func (h *Handler) AdminCategoryGet(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.CategoryGet(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.FromError(w, r, err, "category not found")
		return
	}
	response.OK(w, c)
}

// AdminCategoryCreate godoc
// @Summary  Create a category (admin)
// @Tags     taxonomy
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body destination.CategoryInput true "Category"
// @Success  201 {object} response.Envelope
// @Router   /v1/admin/categories [post]
func (h *Handler) AdminCategoryCreate(w http.ResponseWriter, r *http.Request) {
	var in CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	id, err := h.svc.CategoryCreate(r.Context(), in)
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.Created(w, map[string]string{"id": id})
}

// AdminCategoryUpdate godoc
// @Summary  Update a category (admin)
// @Tags     taxonomy
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id   path string                true "Category ID"
// @Param    body body destination.CategoryInput true "Category"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/categories/{id} [put]
func (h *Handler) AdminCategoryUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in CategoryInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.CategoryUpdate(r.Context(), id, in); err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

// AdminCategoryDelete godoc
// @Summary  Delete a category (admin)
// @Tags     taxonomy
// @Security BearerAuth
// @Param    id path string true "Category ID"
// @Success  204
// @Router   /v1/admin/categories/{id} [delete]
func (h *Handler) AdminCategoryDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.CategoryDelete(r.Context(), chi.URLParam(r, "id")); err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.NoContent(w)
}

// ─── Admin: regions ─────────────────────────────────────────────

// AdminRegionGet godoc
// @Summary  Region detail (admin)
// @Tags     taxonomy
// @Produce  json
// @Param    id path string true "Region ID"
// @Success  200 {object} response.Envelope{data=destination.Region}
// @Router   /v1/regions/{id} [get]
func (h *Handler) AdminRegionGet(w http.ResponseWriter, r *http.Request) {
	reg, err := h.svc.RegionGet(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.FromError(w, r, err, "region not found")
		return
	}
	response.OK(w, reg)
}

// AdminRegionCreate godoc
// @Summary  Create a region (admin)
// @Tags     taxonomy
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body destination.RegionInput true "Region"
// @Success  201 {object} response.Envelope
// @Router   /v1/admin/regions [post]
func (h *Handler) AdminRegionCreate(w http.ResponseWriter, r *http.Request) {
	var in RegionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	id, err := h.svc.RegionCreate(r.Context(), in)
	if err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.Created(w, map[string]string{"id": id})
}

// AdminRegionUpdate godoc
// @Summary  Update a region (admin)
// @Tags     taxonomy
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id   path string              true "Region ID"
// @Param    body body destination.RegionInput true "Region"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/regions/{id} [put]
func (h *Handler) AdminRegionUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in RegionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if err := h.svc.RegionUpdate(r.Context(), id, in); err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

// AdminRegionDelete godoc
// @Summary  Delete a region (admin)
// @Tags     taxonomy
// @Security BearerAuth
// @Param    id path string true "Region ID"
// @Success  204
// @Router   /v1/admin/regions/{id} [delete]
func (h *Handler) AdminRegionDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RegionDelete(r.Context(), chi.URLParam(r, "id")); err != nil {
		response.FromError(w, r, err, "")
		return
	}
	response.NoContent(w)
}

// ─── helpers ────────────────────────────────────────────────────

// splitFloats parses a comma-separated list of floats, skipping invalid parts.
// Kept verbatim from the original handler so bbox parsing behaves identically.
func splitFloats(s string) []float64 {
	out := []float64{}
	current := ""
	for _, ch := range s {
		if ch == ',' {
			if v, err := strconv.ParseFloat(current, 64); err == nil {
				out = append(out, v)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		if v, err := strconv.ParseFloat(current, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}
