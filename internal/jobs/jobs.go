// Package jobs — background workers. Currently:
//
//   • advisory cleaner: expires advisories past their TTL every 5 min
//   • weather refresh: keeps the top destinations' weather snapshots warm
//
// Each job runs in a goroutine started by Start(). No external scheduler.
package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/internal/ws"
)

func Start(pool *pgxpool.Pool, hub *ws.Hub, log *slog.Logger) {
	go advisoryCleaner(pool, hub, log)
	log.Info("jobs · started", slog.String("worker", "advisory-cleaner"))
}

// Expire stale advisories every 5 minutes. Could also be a Postgres CRON.
func advisoryCleaner(pool *pgxpool.Pool, hub *ws.Hub, log *slog.Logger) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var cleared int
		_ = pool.QueryRow(ctx, `
			WITH cleared AS (
			  DELETE FROM advisories WHERE effective_to <= now() - INTERVAL '1 hour'
			  RETURNING 1
			)
			SELECT COUNT(*) FROM cleared
		`).Scan(&cleared)
		cancel()
		if cleared > 0 {
			log.Info("advisories cleared", slog.Int("n", cleared))
			hub.Broadcast(map[string]any{"type": "advisory.cleared", "count": cleared})
		}
	}
}
