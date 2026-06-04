// Package provider — houseboats, shikara, guides, ponies, cabs, helis.
package provider

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type Provider struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Name             string   `json:"name"`
	JktdcRegNo       *string  `json:"jktdc_reg_no,omitempty"`
	Verified         bool     `json:"verified"`
	BaseLocationText *string  `json:"base_location_text,omitempty"`
	Languages        []string `json:"languages,omitempty"`
	Rating           float64  `json:"rating"`
	ReviewCount      int      `json:"review_count"`
	Phone            *string  `json:"phone,omitempty"`
	Whatsapp         *string  `json:"whatsapp,omitempty"`
	Capacity         *int     `json:"capacity,omitempty"`
	Amenities        []string `json:"amenities,omitempty"`
	PriceINR         int      `json:"price_inr"`
	PriceUnit        string   `json:"price_unit"`
	Cancellation     *string  `json:"cancellation,omitempty"`
	Description      *string  `json:"description,omitempty"`
	YearsHosting     *int     `json:"years_hosting,omitempty"`
	ResponseTimeMin  *int     `json:"response_time_min,omitempty"`
}

const providerCols = `id::text, type, name, jktdc_reg_no, verified, base_location_text,
	languages, rating, review_count, phone, whatsapp, capacity, amenities, price_inr, price_unit,
	cancellation, description, years_hosting, response_time_min`

func scanProvider(row interface {
	Scan(dest ...any) error
}) (Provider, error) {
	var p Provider
	err := row.Scan(&p.ID, &p.Type, &p.Name, &p.JktdcRegNo, &p.Verified, &p.BaseLocationText,
		&p.Languages, &p.Rating, &p.ReviewCount, &p.Phone, &p.Whatsapp, &p.Capacity, &p.Amenities,
		&p.PriceINR, &p.PriceUnit, &p.Cancellation, &p.Description, &p.YearsHosting, &p.ResponseTimeMin)
	return p, err
}

// List godoc
// @Summary  List providers
// @Tags     providers
// @Produce  json
// @Param    type     query string false "Filter by type (houseboat, shikara, guide, pony, cab, heli)"
// @Param    verified query bool   false "Only verified providers"
// @Success  200 {object} response.Envelope{data=[]provider.Provider}
// @Router   /v1/providers [get]
func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	typ := q.Get("type")
	verifiedOnly := q.Get("verified") == "true"

	rows, err := s.pool.Query(r.Context(), `
		SELECT `+providerCols+`
		FROM providers
		WHERE ($1 = '' OR type = $1)
		  AND (NOT $2 OR verified = true)
		ORDER BY verified DESC, rating DESC
	`, typ, verifiedOnly)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := make([]Provider, 0)
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, p)
	}
	response.OK(w, out)
}

// Get godoc
// @Summary  Get a provider
// @Tags     providers
// @Produce  json
// @Param    id path string true "Provider ID"
// @Success  200 {object} response.Envelope{data=provider.Provider}
// @Failure  404 {object} response.Envelope
// @Router   /v1/providers/{id} [get]
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := scanProvider(s.pool.QueryRow(r.Context(), `
		SELECT `+providerCols+` FROM providers WHERE id = $1
	`, id))
	if err != nil {
		response.NotFound(w, "provider not found")
		return
	}
	response.OK(w, p)
}

// AdminVerify godoc
// @Summary  Verify a provider (admin)
// @Tags     admin-providers
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Provider ID"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/providers/{id}/verify [post]
func (s *Service) AdminVerify(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `UPDATE providers SET verified = true WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]any{"verified": true})
}

// ─── Admin CRUD ────────────────────────────────────────────────

// ProviderInput is the writable shape accepted from the admin panel. Optional
// text fields are normalised to NULL when blank; numeric optionals arrive as
// pointers so an omitted value stays NULL rather than 0.
type ProviderInput struct {
	Type             string   `json:"type"`
	Name             string   `json:"name"`
	JktdcRegNo       string   `json:"jktdc_reg_no"`
	Verified         bool     `json:"verified"`
	BaseLocationText string   `json:"base_location_text"`
	Languages        []string `json:"languages"`
	Rating           float64  `json:"rating"`
	ReviewCount      int      `json:"review_count"`
	Phone            string   `json:"phone"`
	Whatsapp         string   `json:"whatsapp"`
	Capacity         *int     `json:"capacity"`
	Amenities        []string `json:"amenities"`
	PriceINR         int      `json:"price_inr"`
	PriceUnit        string   `json:"price_unit"`
	Cancellation     string   `json:"cancellation"`
	Description      string   `json:"description"`
	YearsHosting     *int     `json:"years_hosting"`
	ResponseTimeMin  *int     `json:"response_time_min"`
}

// decode reads + validates the body and applies sane defaults for the NOT NULL
// columns (type, price_unit) so the admin's partial payloads insert cleanly.
func (in *ProviderInput) decode(r *http.Request) error {
	if err := json.NewDecoder(r.Body).Decode(in); err != nil {
		return err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Type == "" {
		in.Type = "guide"
	}
	if in.PriceUnit == "" {
		in.PriceUnit = "per-person"
	}
	return nil
}

// AdminCreate godoc
// @Summary  Create a provider (admin)
// @Tags     admin-providers
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body provider.ProviderInput true "Provider"
// @Success  201 {object} response.Envelope
// @Failure  400 {object} response.Envelope
// @Router   /v1/admin/providers [post]
func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var in ProviderInput
	if err := in.decode(r); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if in.Name == "" {
		response.BadRequest(w, "name required")
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO providers (type, name, jktdc_reg_no, verified, base_location_text,
			languages, rating, review_count, phone, whatsapp, capacity, amenities,
			price_inr, price_unit, cancellation, description, years_hosting, response_time_min)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		RETURNING id::text
	`, in.Type, in.Name, nz(in.JktdcRegNo), in.Verified, nz(in.BaseLocationText),
		in.Languages, in.Rating, in.ReviewCount, nz(in.Phone), nz(in.Whatsapp), in.Capacity, in.Amenities,
		in.PriceINR, in.PriceUnit, nz(in.Cancellation), nz(in.Description), in.YearsHosting, in.ResponseTimeMin).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.Created(w, map[string]string{"id": id})
}

// AdminUpdate godoc
// @Summary  Update a provider (admin)
// @Tags     admin-providers
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id   path string         true "Provider ID"
// @Param    body body provider.ProviderInput true "Provider"
// @Success  200 {object} response.Envelope
// @Failure  400 {object} response.Envelope
// @Failure  404 {object} response.Envelope
// @Router   /v1/admin/providers/{id} [put]
func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in ProviderInput
	if err := in.decode(r); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if in.Name == "" {
		response.BadRequest(w, "name required")
		return
	}
	tag, err := s.pool.Exec(r.Context(), `
		UPDATE providers SET
			type = $2, name = $3, jktdc_reg_no = $4, verified = $5, base_location_text = $6,
			languages = $7, rating = $8, review_count = $9, phone = $10, whatsapp = $11,
			capacity = $12, amenities = $13, price_inr = $14, price_unit = $15,
			cancellation = $16, description = $17, years_hosting = $18, response_time_min = $19
		WHERE id = $1
	`, id, in.Type, in.Name, nz(in.JktdcRegNo), in.Verified, nz(in.BaseLocationText),
		in.Languages, in.Rating, in.ReviewCount, nz(in.Phone), nz(in.Whatsapp), in.Capacity, in.Amenities,
		in.PriceINR, in.PriceUnit, nz(in.Cancellation), nz(in.Description), in.YearsHosting, in.ResponseTimeMin)
	if err != nil {
		response.Internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		response.NotFound(w, "provider not found")
		return
	}
	response.OK(w, map[string]string{"id": id})
}

// AdminDelete godoc
// @Summary  Delete a provider (admin)
// @Tags     admin-providers
// @Security BearerAuth
// @Param    id path string true "Provider ID"
// @Success  204
// @Router   /v1/admin/providers/{id} [delete]
func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM providers WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}

// nz returns nil for blank strings so optional text columns store NULL.
func nz(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
