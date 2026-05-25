// Package permit — J&K permit registry + check-for-trip helper.
package permit

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// GET /v1/permits
func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, name, required, office, processing_days,
		       cost_inr, validity, status, notes, official_url
		FROM permits ORDER BY id
	`)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, req, office, days, cost, validity, status, notes, url string
		_ = rows.Scan(&id, &name, &req, &office, &days, &cost, &validity, &status, &notes, &url)
		out = append(out, map[string]any{
			"id": id, "name": name, "required": req, "office": office,
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
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	required := []string{}
	for rows.Next() {
		var p string
		_ = rows.Scan(&p)
		required = append(required, p)
	}

	if len(required) == 0 {
		response.OK(w, []any{})
		return
	}

	prows, err := s.pool.Query(r.Context(), `
		SELECT id::text, name, required, office, processing_days, cost_inr, validity, status, notes, official_url
		FROM permits WHERE LOWER(SUBSTRING(name FROM 1 FOR 3)) = ANY($1::TEXT[])
	`, lcShort(required))
	if err != nil { response.Internal(w, err); return }
	defer prows.Close()

	out := []map[string]any{}
	for prows.Next() {
		var id, name, req, office, days, cost, validity, status, notes, url string
		_ = prows.Scan(&id, &name, &req, &office, &days, &cost, &validity, &status, &notes, &url)
		out = append(out, map[string]any{
			"id": id, "name": name, "required": req, "office": office,
			"processing_days": days, "cost_inr": cost, "validity": validity,
			"status": status, "notes": notes, "official_url": url,
		})
	}
	response.OK(w, out)
}

func lcShort(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if len(s) >= 3 { out = append(out, strings.ToLower(s[:3])) }
	}
	return out
}
