// Package cultural — food / festivals / crafts / etiquette content.
package cultural

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// GET /v1/cultural/food
func (s *Service) Food(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text,
		       COALESCE(name_local->>'en', name) AS name,
		       COALESCE(name_local->>'ur', '') AS name_urdu,
		       COALESCE(name_local->>'ks', '') AS name_kashmiri,
		       COALESCE((details->>'vegetarian')::boolean, false) AS vegetarian,
		       COALESCE(description, '') AS description,
		       COALESCE(details->>'where_to_try', '') AS where_to_try,
		       COALESCE(details->>'price_range', '') AS price_range
		FROM cultural_items WHERE type = 'dish'
		ORDER BY name
	`)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, urdu, ks, desc, where, price string
		var veg bool
		_ = rows.Scan(&id, &name, &urdu, &ks, &veg, &desc, &where, &price)
		out = append(out, map[string]any{
			"id": id, "name": name, "name_urdu": urdu, "name_kashmiri": ks,
			"vegetarian": veg, "description": desc,
			"where_to_try": where, "price_range": price,
		})
	}
	response.OK(w, out)
}

// GET /v1/cultural/festivals
func (s *Service) Festivals(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, name,
		       COALESCE((details->>'month')::int, 0),
		       COALESCE(details->>'duration', ''),
		       COALESCE(description, ''),
		       COALESCE(details->>'region', '')
		FROM cultural_items WHERE type = 'festival'
		ORDER BY (details->>'month')::int NULLS LAST
	`)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, dur, desc, region string
		var month int
		_ = rows.Scan(&id, &name, &month, &dur, &desc, &region)
		out = append(out, map[string]any{
			"id": id, "name": name, "month": month,
			"duration": dur, "description": desc, "region": region,
		})
	}
	response.OK(w, out)
}

// GET /v1/cultural/crafts
func (s *Service) Crafts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, name,
		       COALESCE(details->>'origin', ''),
		       COALESCE(details->>'price', ''),
		       COALESCE(description, '')
		FROM cultural_items WHERE type = 'craft'
		ORDER BY name
	`)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, origin, price, desc string
		_ = rows.Scan(&id, &name, &origin, &price, &desc)
		out = append(out, map[string]any{
			"id": id, "name": name, "origin": origin, "price": price, "description": desc,
		})
	}
	response.OK(w, out)
}

// GET /v1/cultural/etiquette
func (s *Service) Etiquette(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT COALESCE(details->>'category', ''), name, COALESCE(description, '')
		FROM cultural_items WHERE type = 'etiquette'
		ORDER BY (details->>'category')
	`)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var cat, title, body string
		_ = rows.Scan(&cat, &title, &body)
		out = append(out, map[string]any{"category": cat, "title": title, "body": body})
	}
	response.OK(w, out)
}

// POST /v1/admin/cultural — admin create.
func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Type        string          `json:"type"`
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Details     json.RawMessage `json:"details"`
		NameLocal   json.RawMessage `json:"name_local"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `
		INSERT INTO cultural_items (type, name, description, details, name_local)
		VALUES ($1, $2, $3, $4, $5)
	`, body.Type, body.Name, body.Description, body.Details, body.NameLocal); err != nil {
		response.Internal(w, err); return
	}
	response.Created(w, map[string]bool{"created": true})
}

// ─── Admin CRUD ────────────────────────────────────────────────

func (s *Service) AdminGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var item struct {
		ID          string          `json:"id"`
		Type        string          `json:"type"`
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Details     json.RawMessage `json:"details"`
		NameLocal   json.RawMessage `json:"name_local"`
	}
	err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, type, name, COALESCE(description, ''),
		       COALESCE(details, '{}'::jsonb), COALESCE(name_local, '{}'::jsonb)
		FROM cultural_items WHERE id = $1
	`, id).Scan(&item.ID, &item.Type, &item.Name, &item.Description, &item.Details, &item.NameLocal)
	if err != nil {
		response.Internal(w, err); return
	}
	response.OK(w, item)
}

// AdminCreateFor returns a handler scoped to a cultural type.
func (s *Service) AdminCreateFor(ctype string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Details     json.RawMessage `json:"details"`
			NameLocal   json.RawMessage `json:"name_local"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.BadRequest(w, "invalid body"); return
		}
		var id string
		err := s.pool.QueryRow(r.Context(), `
			INSERT INTO cultural_items (type, name, description, details, name_local)
			VALUES ($1, $2, $3, $4, $5) RETURNING id::text
		`, ctype, body.Name, body.Description, body.Details, body.NameLocal).Scan(&id)
		if err != nil {
			response.Internal(w, err); return
		}
		response.Created(w, map[string]string{"id": id})
	}
}

func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Details     json.RawMessage `json:"details"`
		NameLocal   json.RawMessage `json:"name_local"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE cultural_items SET name=$1, description=$2, details=$3, name_local=$4
		WHERE id=$5
	`, body.Name, body.Description, body.Details, body.NameLocal, id)
	if err != nil {
		response.Internal(w, err); return
	}
	response.OK(w, map[string]string{"updated": id})
}

func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM cultural_items WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err); return
	}
	response.NoContent(w)
}
