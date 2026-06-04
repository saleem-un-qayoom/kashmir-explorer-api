// Package search — semantic + filter search across destinations + treks.
//
// Hybrid ranker: 70% pgvector cosine similarity on the query embedding,
// 30% pg_trgm fuzzy match on name/tagline. Filters (region, max altitude,
// permit) apply on top. Returns mixed result types so the mobile Search
// screen renders destinations + treks in one list.
package search

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/internal/clients"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool *pgxpool.Pool
	emb  *clients.Embeddings
}

func NewService(pool *pgxpool.Pool, embKey string) *Service {
	return &Service{pool: pool, emb: clients.NewEmbeddings(embKey)}
}

type Result struct {
	Kind       string  `json:"kind"` // 'destination' | 'trek'
	ID         string  `json:"id"`
	Slug       string  `json:"slug"`
	Name       string  `json:"name"`
	Tagline    *string `json:"tagline,omitempty"`
	Score      float64 `json:"score"`
	District   *string `json:"district,omitempty"`
	AltitudeM  *int    `json:"altitude_m,omitempty"`
	Difficulty *string `json:"difficulty,omitempty"`
	Days       *int    `json:"days,omitempty"`
}

// SearchResponse is the search payload (OpenAPI/codegen model).
type SearchResponse struct {
	Query     string   `json:"query"`
	Results   []Result `json:"results"`
	VectorHit bool     `json:"vector_hit"`
}

// Search godoc
// @Summary  Hybrid semantic + fuzzy search (destinations + treks)
// @Tags     search
// @Produce  json
// @Param    q         query string true  "Query text"
// @Param    region    query string false "Region slug filter"
// @Param    max_alt   query int    false "Max altitude (m) filter"
// @Param    permit_ok query bool   false "Include permit-required results (default true)"
// @Param    limit     query int    false "Max results (default 20, max 50)"
// @Success  200 {object} response.Envelope{data=search.SearchResponse}
// @Failure  400 {object} response.Envelope
// @Router   /v1/search [get]
func (s *Service) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	if query == "" {
		response.BadRequest(w, "q (query) required")
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	maxAlt, _ := strconv.Atoi(q.Get("max_alt"))
	region := q.Get("region")
	permitOk := q.Get("permit_ok") != "false"

	// Get the query embedding (may fail → fall back to trigram only).
	var vector string
	if vec, err := s.emb.EmbedQuery(r.Context(), query); err == nil {
		vector = clients.PgvectorString(vec)
	}

	results := make([]Result, 0, limit)
	if vector != "" {
		rows, err := s.pool.Query(r.Context(), `
			WITH dest AS (
			  SELECT 'destination'::text AS kind, d.id::text, d.slug, d.name, d.tagline,
			         d.district, d.altitude_m, NULL::text AS difficulty, NULL::int AS days,
			         (1 - (d.embedding <=> $1::vector)) * 0.7
			           + similarity(LOWER(d.name || ' ' || COALESCE(d.tagline, '')), LOWER($2::text)) * 0.3
			         AS score
			  FROM destinations d
			  WHERE d.is_published = true AND d.embedding IS NOT NULL
			    AND ($3 = '' OR EXISTS (SELECT 1 FROM regions r WHERE r.id = d.region_id AND r.slug = $3))
			    AND ($4 = 0 OR d.altitude_m IS NULL OR d.altitude_m <= $4)
			    AND ($5 OR COALESCE(array_length(d.permits, 1), 0) = 0)
			),
			trek AS (
			  SELECT 'trek'::text AS kind, t.id::text, t.slug, t.name, t.tagline,
			         NULL::text AS district, t.max_altitude_m AS altitude_m,
			         t.difficulty, t.duration_days AS days,
			         (1 - (t.embedding <=> $1::vector)) * 0.7
			           + similarity(LOWER(t.name || ' ' || COALESCE(t.tagline, '')), LOWER($2::text)) * 0.3
			         AS score
			  FROM treks t
			  WHERE t.is_published = true AND t.embedding IS NOT NULL
			    AND ($4 = 0 OR t.max_altitude_m <= $4)
			    AND ($5 OR COALESCE(array_length(t.permits, 1), 0) = 0)
			)
			SELECT kind, id, slug, name, tagline, district, altitude_m, difficulty, days, score
			FROM (SELECT * FROM dest UNION ALL SELECT * FROM trek) u
			WHERE score > 0.20
			ORDER BY score DESC
			LIMIT $6
		`, vector, query, region, maxAlt, permitOk, limit)
		if err != nil {
			response.Internal(w, err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var item Result
			if err := rows.Scan(&item.Kind, &item.ID, &item.Slug, &item.Name, &item.Tagline,
				&item.District, &item.AltitudeM, &item.Difficulty, &item.Days, &item.Score); err == nil {
				results = append(results, item)
			}
		}
	} else {
		// Trigram-only fallback.
		rows, err := s.pool.Query(r.Context(), `
			SELECT 'destination', id::text, slug, name, tagline, district, altitude_m,
			       similarity(LOWER(name || ' ' || COALESCE(tagline, '')), LOWER($1)) AS score
			FROM destinations WHERE is_published = true
			  AND similarity(LOWER(name || ' ' || COALESCE(tagline, '')), LOWER($1)) > 0.1
			ORDER BY score DESC
			LIMIT $2
		`, query, limit)
		if err != nil {
			response.Internal(w, err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var item Result
			if err := rows.Scan(&item.Kind, &item.ID, &item.Slug, &item.Name, &item.Tagline,
				&item.District, &item.AltitudeM, &item.Score); err != nil {
				response.Internal(w, err)
				return
			}
			results = append(results, item)
		}
	}

	response.OK(w, map[string]any{
		"query":      query,
		"results":    results,
		"vector_hit": vector != "",
	})
}

/* ─── Reindex job ───────────────────────────────────────── */

// Reindex godoc
// @Summary  Re-embed rows missing a vector (admin)
// @Tags     admin-search
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/reindex [post]
func (s *Service) Reindex(w http.ResponseWriter, r *http.Request) {
	dests, err := s.indexBatch(r.Context(), "destinations", `
		SELECT id::text, name || ' · ' || COALESCE(district, '') || ' · ' ||
		       COALESCE(tagline, '') || ' · ' || COALESCE(uniqueness, '')
		FROM destinations WHERE is_published = true AND embedding IS NULL
		LIMIT 50
	`)
	if err != nil {
		response.Internal(w, fmt.Errorf("dest index: %w", err))
		return
	}

	treks, err := s.indexBatch(r.Context(), "treks", `
		SELECT id::text, name || ' · ' || difficulty || ' · ' ||
		       duration_days || ' day · ' || COALESCE(tagline, '') || ' · ' ||
		       COALESCE(uniqueness, '')
		FROM treks WHERE is_published = true AND embedding IS NULL
		LIMIT 50
	`)
	if err != nil {
		response.Internal(w, fmt.Errorf("trek index: %w", err))
		return
	}

	response.OK(w, map[string]any{
		"destinations_indexed": dests,
		"treks_indexed":        treks,
	})
}

func (s *Service) indexBatch(ctx context.Context, table, query string) (int, error) {
	type row struct{ id, text string }
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var batch []row
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.text); err != nil {
			return 0, err
		}
		batch = append(batch, item)
	}
	if len(batch) == 0 {
		return 0, nil
	}

	texts := make([]string, len(batch))
	for i, item := range batch {
		texts[i] = item.text
	}
	vecs, err := s.emb.EmbedDocs(ctx, texts)
	if err != nil {
		return 0, err
	}

	for i, item := range batch {
		_, _ = s.pool.Exec(ctx,
			fmt.Sprintf(`UPDATE %s SET embedding = $1::vector WHERE id = $2::uuid`, table),
			clients.PgvectorString(vecs[i]), item.id)
	}
	return len(batch), nil
}
