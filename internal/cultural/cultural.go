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

// Cultural read models (OpenAPI/codegen; handlers emit these exact fields).
type Dish struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	NameUrdu     string `json:"name_urdu"`
	NameKashmiri string `json:"name_kashmiri"`
	Vegetarian   bool   `json:"vegetarian"`
	Description  string `json:"description"`
	WhereToTry   string `json:"where_to_try"`
	PriceRange   string `json:"price_range"`
}

type Festival struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Month       int    `json:"month"`
	Duration    string `json:"duration"`
	Description string `json:"description"`
	Region      string `json:"region"`
}

type Craft struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Origin      string `json:"origin"`
	Price       string `json:"price"`
	Description string `json:"description"`
}

type EtiquetteTip struct {
	Category string `json:"category"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

// Food godoc
// @Summary  List Wazwan dishes
// @Tags     cultural
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]cultural.Dish}
// @Router   /v1/cultural/food [get]
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
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, urdu, ks, desc, where, price string
		var veg bool
		if err := rows.Scan(&id, &name, &urdu, &ks, &veg, &desc, &where, &price); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{
			"id": id, "name": name, "name_urdu": urdu, "name_kashmiri": ks,
			"vegetarian": veg, "description": desc,
			"where_to_try": where, "price_range": price,
		})
	}
	response.OK(w, out)
}

// Festivals godoc
// @Summary  List festivals
// @Tags     cultural
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]cultural.Festival}
// @Router   /v1/cultural/festivals [get]
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
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, dur, desc, region string
		var month int
		if err := rows.Scan(&id, &name, &month, &dur, &desc, &region); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{
			"id": id, "name": name, "month": month,
			"duration": dur, "description": desc, "region": region,
		})
	}
	response.OK(w, out)
}

// Crafts godoc
// @Summary  List handicrafts
// @Tags     cultural
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]cultural.Craft}
// @Router   /v1/cultural/crafts [get]
func (s *Service) Crafts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, name,
		       COALESCE(details->>'origin', ''),
		       COALESCE(details->>'price', ''),
		       COALESCE(description, '')
		FROM cultural_items WHERE type = 'craft'
		ORDER BY name
	`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, origin, price, desc string
		if err := rows.Scan(&id, &name, &origin, &price, &desc); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{
			"id": id, "name": name, "origin": origin, "price": price, "description": desc,
		})
	}
	response.OK(w, out)
}

// Etiquette godoc
// @Summary  List etiquette tips
// @Tags     cultural
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]cultural.EtiquetteTip}
// @Router   /v1/cultural/etiquette [get]
func (s *Service) Etiquette(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT COALESCE(details->>'category', ''), name, COALESCE(description, '')
		FROM cultural_items WHERE type = 'etiquette'
		ORDER BY (details->>'category')
	`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var cat, title, body string
		if err := rows.Scan(&cat, &title, &body); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{"category": cat, "title": title, "body": body})
	}
	response.OK(w, out)
}

// Cultural admin doc-models (OpenAPI/codegen).
type CulturalItem struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Details     json.RawMessage `json:"details"`
	NameLocal   json.RawMessage `json:"name_local"`
}
type CulturalInput struct {
	Type        string          `json:"type,omitempty"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Details     json.RawMessage `json:"details"`
	NameLocal   json.RawMessage `json:"name_local"`
}

// AdminCreate godoc
// @Summary  Create a cultural item (admin, generic)
// @Tags     admin-cultural
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body cultural.CulturalInput true "Cultural item"
// @Success  201 {object} response.Envelope
// @Router   /v1/admin/cultural [post]
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
		response.Internal(w, err)
		return
	}
	response.Created(w, map[string]bool{"created": true})
}

// ─── Admin CRUD ────────────────────────────────────────────────

// AdminGet godoc
// @Summary  Get a cultural item by ID (per type)
// @Tags     cultural
// @Produce  json
// @Param    id path string true "Item ID"
// @Success  200 {object} response.Envelope{data=cultural.CulturalItem}
// @Router   /v1/cultural/food/{id} [get]
// @Router   /v1/cultural/festivals/{id} [get]
// @Router   /v1/cultural/crafts/{id} [get]
// @Router   /v1/cultural/etiquette/{id} [get]
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
		response.Internal(w, err)
		return
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
			response.BadRequest(w, "invalid body")
			return
		}
		var id string
		err := s.pool.QueryRow(r.Context(), `
			INSERT INTO cultural_items (type, name, description, details, name_local)
			VALUES ($1, $2, $3, $4, $5) RETURNING id::text
		`, ctype, body.Name, body.Description, body.Details, body.NameLocal).Scan(&id)
		if err != nil {
			response.Internal(w, err)
			return
		}
		response.Created(w, map[string]string{"id": id})
	}
}

// AdminUpdate godoc
// @Summary  Update a cultural item (admin, per type)
// @Tags     admin-cultural
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id   path string              true "Item ID"
// @Param    body body cultural.CulturalInput true "Cultural item"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/cultural/food/{id} [put]
// @Router   /v1/admin/cultural/festivals/{id} [put]
// @Router   /v1/admin/cultural/crafts/{id} [put]
// @Router   /v1/admin/cultural/etiquette/{id} [put]
func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Details     json.RawMessage `json:"details"`
		NameLocal   json.RawMessage `json:"name_local"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE cultural_items SET name=$1, description=$2, details=$3, name_local=$4
		WHERE id=$5
	`, body.Name, body.Description, body.Details, body.NameLocal, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

// AdminDelete godoc
// @Summary  Delete a cultural item (admin, per type)
// @Tags     admin-cultural
// @Security BearerAuth
// @Param    id path string true "Item ID"
// @Success  204
// @Router   /v1/admin/cultural/food/{id} [delete]
// @Router   /v1/admin/cultural/festivals/{id} [delete]
// @Router   /v1/admin/cultural/crafts/{id} [delete]
// @Router   /v1/admin/cultural/etiquette/{id} [delete]
func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM cultural_items WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}
