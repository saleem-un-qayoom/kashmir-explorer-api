// Package ai — Anthropic Claude proxy with destination RAG.
//
// Two endpoints:
//   POST /v1/ai/plan-trip — single-shot itinerary JSON
//   POST /v1/ai/ask       — streaming SSE chat response with citations
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/internal/clients"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool *pgxpool.Pool
	llm  *clients.Anthropic
}

func NewService(apiKey, model string, pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, llm: clients.NewAnthropic(apiKey, model)}
}

const systemPrompt = `You are Kashmir Explorer's AI travel assistant for the Kashmir region (Jammu & Kashmir, Ladakh).

Style:
- Be specific. Cite real places, altitudes, distances, permits.
- Be honest about limits — say "verify before travelling" when conditions vary.
- Mention Inner Line Permits for Gurez/Bangus/Lolab when relevant.
- For trek questions, mention AMS risk above 3,500m and acclimatisation.
- Be culturally sensitive: refer to Wazwan, Pheran, Kahwa by their local names.
- Currency is INR (₹). Distances in km, altitudes in metres (feet in parens for treks).

Reference data passed in <context> is fresh from our database.`

type planTripReq struct {
	Days     int      `json:"days"`
	Month    int      `json:"month"`
	Persona  []string `json:"persona"`
	Base     string   `json:"base"`
	Budget   string   `json:"budget"`
	MaxAltM  int      `json:"max_alt_m"`
	PermitOk bool     `json:"permit_ok"`
	VegOnly  bool     `json:"veg_only"`
}

// POST /v1/ai/plan-trip
func (s *Service) PlanTrip(w http.ResponseWriter, r *http.Request) {
	var body planTripReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	dests, err := s.candidateDestinations(ctx, body)
	if err != nil { response.Internal(w, err); return }

	user := fmt.Sprintf(`Plan a %d-day trip to Kashmir for month %d.
Based from: %s. Budget: %s. Max altitude: %dm.
Permit-allowed: %v. Vegetarian-only: %v.
Interests: %s.

<context>
Candidate destinations (from our DB, all currently published):
%s
</context>

Return STRICT JSON in this shape:
{
  "title": "string",
  "days": %d,
  "itinerary": [
    {"day": 1, "title": "string", "stops": ["slug-1", "slug-2"]},
    ...
  ],
  "notes": "1-2 sentences of practical advice"
}

Use ONLY slugs from the candidate list above. JSON only, no preamble.`,
		body.Days, body.Month, body.Base, body.Budget, body.MaxAltM,
		body.PermitOk, body.VegOnly, strings.Join(body.Persona, ", "),
		dests, body.Days)

	text, err := s.llm.Complete(ctx, systemPrompt, []clients.Message{
		{Role: "user", Content: user},
	}, 2000)
	if err != nil { response.Internal(w, err); return }

	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		response.OK(w, map[string]any{"raw": text, "error": "AI returned unparseable JSON"})
		return
	}
	response.OK(w, parsed)
}

type askReq struct {
	DestinationID string `json:"destination_id,omitempty"`
	Question      string `json:"question"`
}

/* ─── Streaming SSE endpoint ────────────────────────────── */

// POST /v1/ai/ask — server-sent events with token-by-token forwarding.
//
// Event types:
//   event: chunk    \n data: {"text":"..."}         — content delta
//   event: citation \n data: {"label":"...","slug":"..."}  — when a known place name is detected
//   event: done     \n data: {}                       — clean close
//   event: error    \n data: {"message":"..."}        — failure
func (s *Service) Ask(w http.ResponseWriter, r *http.Request) {
	var body askReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if body.Question == "" {
		response.BadRequest(w, "question required"); return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		response.Internal(w, fmt.Errorf("streaming unsupported"))
		return
	}

	w.Header().Set("Content-Type",     "text/event-stream")
	w.Header().Set("Cache-Control",    "no-cache")
	w.Header().Set("Connection",       "keep-alive")
	w.Header().Set("X-Accel-Buffering","no") // prevent nginx buffering

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	// Build RAG context.
	var contextText string
	if body.DestinationID != "" {
		contextText, _ = s.singleDestinationContext(ctx, body.DestinationID)
	} else {
		contextText, _ = s.topDestinationsContext(ctx, 8)
	}

	user := fmt.Sprintf(`<context>
%s
</context>

Question: %s

Answer in 2-4 sentences. Be specific. If you cite a place, mention it by name.`,
		contextText, body.Question)

	// Pre-load destination names for citation detection.
	allNames, _ := s.allDestinationNames(ctx)

	// Stream from Anthropic.
	tokens, errs := s.llm.Stream(ctx, systemPrompt, []clients.Message{{Role: "user", Content: user}}, 800)

	var sb strings.Builder
	sentCitations := map[string]bool{}

	for {
		select {
		case <-ctx.Done():
			sse(w, "error", map[string]string{"message": "timeout"})
			flusher.Flush()
			return
		case err := <-errs:
			if err != nil {
				sse(w, "error", map[string]string{"message": err.Error()})
				flusher.Flush()
			}
			return
		case t, ok := <-tokens:
			if !ok {
				// Stream ended cleanly.
				sse(w, "done", map[string]any{"text": sb.String()})
				flusher.Flush()
				return
			}
			sb.WriteString(t)
			sse(w, "chunk", map[string]string{"text": t})

			// Detect new citations on every chunk by scanning the accumulator.
			accum := sb.String()
			for name, slug := range allNames {
				if !sentCitations[slug] && strings.Contains(accum, name) {
					sentCitations[slug] = true
					sse(w, "citation", map[string]string{"label": name, "slug": slug})
				}
			}
			flusher.Flush()
		}
	}
}

func sse(w http.ResponseWriter, event string, data any) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
}

/* ─── Context builders (shared with non-streaming) ─────── */

func (s *Service) candidateDestinations(ctx context.Context, req planTripReq) (string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slug, name, COALESCE(district, ''), COALESCE(altitude_m, 0),
		       COALESCE(tagline, ''), COALESCE(permits, ARRAY[]::TEXT[])
		FROM destinations
		WHERE is_published = true
		  AND ($1 = 0 OR altitude_m IS NULL OR altitude_m <= $1)
		  AND ($2 OR COALESCE(array_length(permits, 1), 0) = 0)
		ORDER BY is_featured DESC, rating DESC
		LIMIT 30
	`, req.MaxAltM, req.PermitOk)
	if err != nil { return "", err }
	defer rows.Close()
	var sb strings.Builder
	for rows.Next() {
		var slug, name, district, tagline string
		var alt int
		var permits []string
		_ = rows.Scan(&slug, &name, &district, &alt, &tagline, &permits)
		permitStr := ""
		if len(permits) > 0 { permitStr = " [permit: " + strings.Join(permits, ",") + "]" }
		fmt.Fprintf(&sb, "- %s (%s) · %s · %dm%s — %s\n", slug, name, district, alt, permitStr, tagline)
	}
	return sb.String(), nil
}

func (s *Service) singleDestinationContext(ctx context.Context, id string) (string, error) {
	var name, district, tagline, uniqueness string
	var alt int
	err := s.pool.QueryRow(ctx, `
		SELECT name, COALESCE(district, ''), COALESCE(altitude_m, 0),
		       COALESCE(tagline, ''), COALESCE(uniqueness, '')
		FROM destinations WHERE id = $1
	`, id).Scan(&name, &district, &alt, &tagline, &uniqueness)
	if err != nil { return "", err }
	return fmt.Sprintf("%s · %s · %dm\nTagline: %s\nWhy unique: %s\n", name, district, alt, tagline, uniqueness), nil
}

func (s *Service) topDestinationsContext(ctx context.Context, n int) (string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT slug, name, COALESCE(tagline, '') FROM destinations
		 WHERE is_published = true ORDER BY rating DESC LIMIT $1`, n)
	if err != nil { return "", err }
	defer rows.Close()
	var sb strings.Builder
	for rows.Next() {
		var slug, name, tag string
		_ = rows.Scan(&slug, &name, &tag)
		fmt.Fprintf(&sb, "- %s (%s) — %s\n", slug, name, tag)
	}
	return sb.String(), nil
}

func (s *Service) allDestinationNames(ctx context.Context) (map[string]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT slug, name FROM destinations WHERE is_published = true`)
	if err != nil { return nil, err }
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var slug, name string
		_ = rows.Scan(&slug, &name)
		out[name] = slug
	}
	return out, nil
}

/* ─── New: image-to-trip (Claude vision) ──────────────── */

type identifyReq struct {
	ImageBase64 string `json:"image_base64"`
	MediaType   string `json:"media_type"` // image/jpeg | image/png | image/webp
}

// POST /v1/ai/identify-place — user uploads a photo, Claude tries to
// identify the location and returns candidate destination slugs.
func (s *Service) IdentifyPlace(w http.ResponseWriter, r *http.Request) {
	var body identifyReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if body.ImageBase64 == "" {
		response.BadRequest(w, "image_base64 required"); return
	}
	if body.MediaType == "" { body.MediaType = "image/jpeg" }

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	corpus, _ := s.topDestinationsContext(ctx, 40)
	prompt := fmt.Sprintf(`Look at the photo. Is it Kashmir? If so, what destination?

Choose ONLY from this list of known destinations:
%s

Reply in JSON: {"is_kashmir": true|false, "best_guess_slug": "slug-or-null", "alternatives": ["slug2","slug3"], "confidence": 0..1, "reasoning": "1 sentence"}`,
		corpus)

	text, err := s.llm.CompleteWithImage(ctx, systemPrompt, prompt, body.ImageBase64, body.MediaType, 600)
	if err != nil { response.Internal(w, err); return }

	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		response.OK(w, map[string]any{"raw": text})
		return
	}
	response.OK(w, parsed)
}
