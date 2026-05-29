// Package permit — J&K permit registry + check-for-trip helper.
package permit

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

// GET /v1/permits
func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, slug, name, required, office, processing_days,
		       cost_inr, validity, status, notes, official_url
		FROM permits ORDER BY id
	`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, slug, name, req, office, days, cost, validity, status, notes, url string
		if err := rows.Scan(&id, &slug, &name, &req, &office, &days, &cost, &validity, &status, &notes, &url); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{
			"id": id, "slug": slug, "name": name, "required": req, "office": office,
			"processing_days": days, "cost_inr": cost, "validity": validity,
			"status": status, "notes": notes, "official_url": url,
		})
	}
	response.OK(w, out)
}

// GET /v1/permits/check?destinations=slug1,slug2
func (s *Service) Check(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("destinations")
	if q == "" {
		response.BadRequest(w, "destinations query param required (csv of slugs)")
		return
	}
	slugs := strings.Split(q, ",")

	rows, err := s.pool.Query(r.Context(), `
		SELECT DISTINCT unnest(d.permits)
		FROM destinations d
		WHERE d.slug = ANY($1) AND COALESCE(array_length(d.permits, 1), 0) > 0
	`, slugs)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	required := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			response.Internal(w, err)
			return
		}
		required = append(required, p)
	}

	if len(required) == 0 {
		response.OK(w, []any{})
		return
	}

	prows, err := s.pool.Query(r.Context(), `
		SELECT id::text, slug, name, required, office, processing_days, cost_inr, validity, status, notes, official_url
		FROM permits WHERE LOWER(SUBSTRING(name FROM 1 FOR 3)) = ANY($1::TEXT[])
	`, lcShort(required))
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer prows.Close()

	out := []map[string]any{}
	for prows.Next() {
		var id, slug, name, req, office, days, cost, validity, status, notes, url string
		_ = prows.Scan(&id, &slug, &name, &req, &office, &days, &cost, &validity, &status, &notes, &url)
		out = append(out, map[string]any{
			"id": id, "slug": slug, "name": name, "required": req, "office": office,
			"processing_days": days, "cost_inr": cost, "validity": validity,
			"status": status, "notes": notes, "official_url": url,
		})
	}
	response.OK(w, out)
}

// ─── Admin CRUD ────────────────────────────────────────────────

func (s *Service) AdminGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var p struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		Required       string  `json:"required"`
		Office         string  `json:"office"`
		ProcessingDays string  `json:"processing_days"`
		CostInr        string  `json:"cost_inr"`
		Validity       string  `json:"validity"`
		Status         string  `json:"status"`
		Notes          *string `json:"notes"`
		OfficialURL    *string `json:"official_url"`
	}
	err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, name, required, office, processing_days,
		       cost_inr, validity, status, notes, official_url
		FROM permits WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Required, &p.Office, &p.ProcessingDays,
		&p.CostInr, &p.Validity, &p.Status, &p.Notes, &p.OfficialURL)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, p)
}

func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Slug           string  `json:"slug"`
		Name           string  `json:"name"`
		Required       string  `json:"required"`
		Office         string  `json:"office"`
		ProcessingDays string  `json:"processing_days"`
		CostInr        string  `json:"cost_inr"`
		Validity       string  `json:"validity"`
		Status         string  `json:"status"`
		Notes          *string `json:"notes"`
		OfficialURL    *string `json:"official_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO permits (slug, name, required, office, processing_days,
		                    cost_inr, validity, status, notes, official_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id::text
	`, in.Slug, in.Name, in.Required, in.Office, in.ProcessingDays,
		in.CostInr, in.Validity, in.Status, in.Notes, in.OfficialURL).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.Created(w, map[string]string{"id": id})
}

func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Slug           string  `json:"slug"`
		Name           string  `json:"name"`
		Required       string  `json:"required"`
		Office         string  `json:"office"`
		ProcessingDays string  `json:"processing_days"`
		CostInr        string  `json:"cost_inr"`
		Validity       string  `json:"validity"`
		Status         string  `json:"status"`
		Notes          *string `json:"notes"`
		OfficialURL    *string `json:"official_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE permits SET slug=$1, name=$2, required=$3, office=$4,
		                  processing_days=$5, cost_inr=$6, validity=$7,
		                  status=$8, notes=$9, official_url=$10
		WHERE id=$11
	`, in.Slug, in.Name, in.Required, in.Office, in.ProcessingDays,
		in.CostInr, in.Validity, in.Status, in.Notes, in.OfficialURL, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM permits WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}

func lcShort(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if len(s) >= 3 {
			out = append(out, strings.ToLower(s[:3]))
		}
	}
	return out
}
