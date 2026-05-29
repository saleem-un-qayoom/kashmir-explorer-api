// Package provider — houseboats, shikara, guides, ponies, cabs, helis.
package provider

import (
	"net/http"

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
	Capacity         *int     `json:"capacity,omitempty"`
	Amenities        []string `json:"amenities,omitempty"`
	PriceINR         int      `json:"price_inr"`
	PriceUnit        string   `json:"price_unit"`
	Cancellation     *string  `json:"cancellation,omitempty"`
	Description      *string  `json:"description,omitempty"`
	YearsHosting     *int     `json:"years_hosting,omitempty"`
	ResponseTimeMin  *int     `json:"response_time_min,omitempty"`
}

// GET /v1/providers ?type=houseboat&verified=true
func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	typ := q.Get("type")
	verifiedOnly := q.Get("verified") == "true"

	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, type, name, jktdc_reg_no, verified, base_location_text,
		       languages, rating, review_count, capacity, amenities, price_inr, price_unit,
		       cancellation, description, years_hosting, response_time_min
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
		var p Provider
		if err := rows.Scan(&p.ID, &p.Type, &p.Name, &p.JktdcRegNo, &p.Verified, &p.BaseLocationText,
			&p.Languages, &p.Rating, &p.ReviewCount, &p.Capacity, &p.Amenities, &p.PriceINR, &p.PriceUnit,
			&p.Cancellation, &p.Description, &p.YearsHosting, &p.ResponseTimeMin); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, p)
	}
	response.OK(w, out)
}

// GET /v1/providers/{id}
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var p Provider
	err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, type, name, jktdc_reg_no, verified, base_location_text,
		       languages, rating, review_count, capacity, amenities, price_inr, price_unit,
		       cancellation, description, years_hosting, response_time_min
		FROM providers WHERE id = $1
	`, id).Scan(&p.ID, &p.Type, &p.Name, &p.JktdcRegNo, &p.Verified, &p.BaseLocationText,
		&p.Languages, &p.Rating, &p.ReviewCount, &p.Capacity, &p.Amenities, &p.PriceINR, &p.PriceUnit,
		&p.Cancellation, &p.Description, &p.YearsHosting, &p.ResponseTimeMin)
	if err != nil {
		response.NotFound(w, "provider not found")
		return
	}
	response.OK(w, p)
}

func (s *Service) AdminVerify(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `UPDATE providers SET verified = true WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]any{"verified": true})
}
