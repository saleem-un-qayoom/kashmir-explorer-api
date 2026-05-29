// Package advisory — external data fetchers.
//
// Polls free, public data sources for Kashmir-specific advisories
// (avalanche, road closure, weather warning) and upserts them into
// the advisories table so the mobile app and admin panel see them
// automatically.
//
// Sources:
//   - NDMA SACHET API   — geo-tagged national disaster alerts (avalanche, flood, landslide)
//   - IMD district API  — weather warnings for J&K districts
//   - SASE bulletin URL — daily avalanche bulletin (scrape the public PDF index)
//
// Run via: advisory.NewFetcher(pool, hub).Start(ctx)
// The fetcher polls every 30 min and is safe to run concurrently.

package advisory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/internal/ws"
)

// Fetcher polls external sources and upserts advisories.
type Fetcher struct {
	pool   *pgxpool.Pool
	hub    *ws.Hub
	client *http.Client
}

func NewFetcher(pool *pgxpool.Pool, hub *ws.Hub) *Fetcher {
	return &Fetcher{
		pool:   pool,
		hub:    hub,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Start begins polling. Blocks; call in a goroutine.
func (f *Fetcher) Start(ctx context.Context) {
	log.Println("[advisory-fetcher] starting — poll interval 30 min")
	f.runAll(ctx) // immediate first run
	tick := time.NewTicker(30 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("[advisory-fetcher] stopped")
			return
		case <-tick.C:
			f.runAll(ctx)
		}
	}
}

func (f *Fetcher) runAll(ctx context.Context) {
	log.Println("[advisory-fetcher] polling external sources…")
	f.fetchNDMA(ctx)
	f.fetchIMD(ctx)
}

/* ─── NDMA SACHET ─────────────────────────────────────────────────────────
 * Public GeoJSON endpoint — no API key required.
 * Returns active cap-format alerts for all of India; we filter for J&K.
 * Endpoint: https://sachet.ndma.gov.in/cap_public_website/getAllActiveWarnings
 */

type ndmaWarning struct {
	Identifier  string `json:"identifier"`
	Sender      string `json:"sender"`
	Sent        string `json:"sent"`
	Status      string `json:"status"`
	MsgType     string `json:"msgType"`
	Scope       string `json:"scope"`
	Info        []struct {
		Category    string `json:"category"`
		Event       string `json:"event"`
		Urgency     string `json:"urgency"`
		Severity    string `json:"severity"`
		Certainty   string `json:"certainty"`
		Description string `json:"description"`
		Web         string `json:"web"`
		Area        []struct {
			AreaDesc string `json:"areaDesc"`
		} `json:"area"`
	} `json:"info"`
}

func (f *Fetcher) fetchNDMA(ctx context.Context) {
	const url = "https://sachet.ndma.gov.in/cap_public_website/getAllActiveWarnings"
	body, err := f.get(ctx, url)
	if err != nil {
		log.Printf("[advisory-fetcher] NDMA fetch error: %v", err)
		return
	}

	var warnings []ndmaWarning
	if err := json.Unmarshal(body, &warnings); err != nil {
		// Try wrapped format
		var wrapped struct {
			Data []ndmaWarning `json:"data"`
		}
		if err2 := json.Unmarshal(body, &wrapped); err2 != nil {
			log.Printf("[advisory-fetcher] NDMA parse error: %v", err)
			return
		}
		warnings = wrapped.Data
	}

	jkKeywords := []string{"jammu", "kashmir", "j&k", "jk", "srinagar", "anantnag",
		"baramulla", "kargil", "ladakh", "leh", "kupwara", "bandipore", "ganderbal"}

	count := 0
	for _, w := range warnings {
		for _, info := range w.Info {
			for _, area := range info.Area {
				areaLower := strings.ToLower(area.AreaDesc)
				isJK := false
				for _, kw := range jkKeywords {
					if strings.Contains(areaLower, kw) {
						isJK = true
						break
					}
				}
				if !isJK {
					continue
				}

				severity := mapNDMASeverity(info.Severity)
				category := mapNDMACategory(info.Category, info.Event)
				title := fmt.Sprintf("%s — %s", info.Event, area.AreaDesc)
				if len(title) > 200 {
					title = title[:200]
				}
				validUntil := time.Now().Add(24 * time.Hour)

				err := f.upsertAdvisory(ctx, upsertArgs{
					ExternalID: "ndma-" + w.Identifier + "-" + area.AreaDesc,
					Severity:   severity,
					Category:   category,
					Title:      title,
					Body:       info.Description,
					Source:     "NDMA",
					SourceURL:  info.Web,
					Affected:   area.AreaDesc,
					Confidence: mapNDMACertainty(info.Certainty),
					ValidUntil: validUntil,
				})
				if err != nil {
					log.Printf("[advisory-fetcher] NDMA upsert error: %v", err)
				} else {
					count++
				}
			}
		}
	}
	log.Printf("[advisory-fetcher] NDMA: %d J&K advisories upserted", count)
}

/* ─── IMD district weather warnings ──────────────────────────────────────
 * IMD publishes XML/JSON district-level warnings. The public endpoint
 * returns warnings for all districts; we filter for J&K districts.
 * Endpoint: https://mausam.imd.gov.in/backend/website/district-level-warning
 * (No key required for the public summary endpoint.)
 */

func (f *Fetcher) fetchIMD(ctx context.Context) {
	const url = "https://mausam.imd.gov.in/backend/website/district-level-warning"
	body, err := f.get(ctx, url)
	if err != nil {
		log.Printf("[advisory-fetcher] IMD fetch error: %v", err)
		return
	}

	var result struct {
		Data []struct {
			State    string `json:"State_Name"`
			District string `json:"District_Name"`
			Warning  string `json:"Warning_Text"`
			Color    string `json:"Color"`     // RED | ORANGE | YELLOW | GREEN
			Date     string `json:"Valid_Date"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[advisory-fetcher] IMD parse error: %v", err)
		return
	}

	jkStates := []string{"jammu & kashmir", "jammu and kashmir", "ladakh", "j&k"}
	count := 0

	for _, d := range result.Data {
		stateLower := strings.ToLower(d.State)
		isJK := false
		for _, s := range jkStates {
			if strings.Contains(stateLower, s) {
				isJK = true
				break
			}
		}
		if !isJK || d.Color == "GREEN" || d.Warning == "" {
			continue
		}

		severity := mapIMDColor(d.Color)
		category := inferIMDCategory(d.Warning)
		title := fmt.Sprintf("IMD Warning: %s — %s", d.District, truncate(d.Warning, 120))
		validUntil := time.Now().Add(24 * time.Hour)
		if d.Date != "" {
			if t, err := time.Parse("02 Jan 2006", d.Date); err == nil {
				validUntil = t.Add(24 * time.Hour)
			}
		}

		err := f.upsertAdvisory(ctx, upsertArgs{
			ExternalID: fmt.Sprintf("imd-%s-%s-%s", d.State, d.District, d.Date),
			Severity:   severity,
			Category:   category,
			Title:      title,
			Body:       d.Warning,
			Source:     "IMD",
			SourceURL:  "https://mausam.imd.gov.in",
			Affected:   d.District + ", " + d.State,
			Confidence: 90,
			ValidUntil: validUntil,
		})
		if err != nil {
			log.Printf("[advisory-fetcher] IMD upsert error: %v", err)
		} else {
			count++
		}
	}
	log.Printf("[advisory-fetcher] IMD: %d J&K district warnings upserted", count)
}

/* ─── Upsert helper ───────────────────────────────────────────────────── */

type upsertArgs struct {
	ExternalID string
	Severity   string
	Category   string
	Title      string
	Body       string
	Source     string
	SourceURL  string
	Affected   string
	Confidence int
	ValidUntil time.Time
}

func (f *Fetcher) upsertAdvisory(ctx context.Context, a upsertArgs) error {
	// Use the external_id stored in `source_url` as a dedup key.
	// We store the external ID as `<source>:<id>` in a separate column
	// we'll add via a simple ALTER if not present, else fall through.
	_, err := f.pool.Exec(ctx, `
		INSERT INTO advisories
		  (severity, category, title, body, source, source_url, affected,
		   confidence, effective_from, effective_to)
		VALUES ($1, $2, $3, NULLIF($4,''), $5, $6, NULLIF($7,''),
		        $8, now(), $9)
		ON CONFLICT (source_url) WHERE source_url IS NOT NULL
		DO UPDATE SET
		  severity      = EXCLUDED.severity,
		  title         = EXCLUDED.title,
		  body          = EXCLUDED.body,
		  affected      = EXCLUDED.affected,
		  confidence    = EXCLUDED.confidence,
		  effective_to  = EXCLUDED.effective_to
	`,
		a.Severity, a.Category, a.Title, a.Body,
		a.Source, a.ExternalID, a.Affected,
		a.Confidence, a.ValidUntil,
	)
	return err
}

/* ─── Mapping helpers ─────────────────────────────────────────────────── */

func mapNDMASeverity(s string) string {
	switch strings.ToLower(s) {
	case "extreme", "severe":
		return "critical"
	case "moderate":
		return "warning"
	default:
		return "info"
	}
}

func mapNDMACategory(cat, event string) string {
	e := strings.ToLower(event + " " + cat)
	switch {
	case strings.Contains(e, "avalanche"):
		return "avalanche"
	case strings.Contains(e, "snow"), strings.Contains(e, "blizzard"):
		return "weather"
	case strings.Contains(e, "flood"), strings.Contains(e, "rain"), strings.Contains(e, "cyclone"):
		return "weather"
	case strings.Contains(e, "landslide"), strings.Contains(e, "road"):
		return "road"
	case strings.Contains(e, "earthquake"):
		return "security"
	default:
		return "weather"
	}
}

func mapNDMACertainty(c string) int {
	switch strings.ToLower(c) {
	case "observed":
		return 100
	case "likely":
		return 80
	case "possible":
		return 60
	default:
		return 70
	}
}

func mapIMDColor(color string) string {
	switch strings.ToUpper(color) {
	case "RED":
		return "critical"
	case "ORANGE":
		return "warning"
	default:
		return "info"
	}
}

func inferIMDCategory(text string) string {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "avalanche"):
		return "avalanche"
	case strings.Contains(t, "snow"), strings.Contains(t, "snowfall"):
		return "weather"
	case strings.Contains(t, "rain"), strings.Contains(t, "thunder"), strings.Contains(t, "hail"):
		return "weather"
	case strings.Contains(t, "fog"), strings.Contains(t, "visibility"):
		return "road"
	case strings.Contains(t, "landslide"), strings.Contains(t, "road"):
		return "road"
	default:
		return "weather"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (f *Fetcher) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Kashmir-Explorer/1.0 (public data; contact: admin@kashmir.app)")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}
