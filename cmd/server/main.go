// Package main — Kashmir Explorer API entry point.
//
// This file is a thin composition root: load config, open the DB pool, build the
// live-broadcast hub/rooms, construct each domain handler, hand them to
// internal/router for wiring, then run the HTTP server lifecycle. All route
// registration lives in internal/router; all request handling lives in the
// per-domain packages.
//
// @title                      Kashmir Explorer API
// @version                    0.1.0
// @description                Backend API for the Kashmir Explorer travel app.
// @BasePath                   /
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
package main

//go:generate swag init -g main.go -o ../../docs --parseDependency --parseInternal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sentry "github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/kashmir-explorer/api/internal/advisory"
	"github.com/kashmir-explorer/api/internal/ai"
	"github.com/kashmir-explorer/api/internal/auth"
	"github.com/kashmir-explorer/api/internal/booking"
	"github.com/kashmir-explorer/api/internal/config"
	"github.com/kashmir-explorer/api/internal/crowd"
	"github.com/kashmir-explorer/api/internal/cultural"
	"github.com/kashmir-explorer/api/internal/destination"
	"github.com/kashmir-explorer/api/internal/groups"
	"github.com/kashmir-explorer/api/internal/image"
	"github.com/kashmir-explorer/api/internal/jobs"
	"github.com/kashmir-explorer/api/internal/permit"
	"github.com/kashmir-explorer/api/internal/photo"
	"github.com/kashmir-explorer/api/internal/provider"
	"github.com/kashmir-explorer/api/internal/report"
	"github.com/kashmir-explorer/api/internal/review"
	"github.com/kashmir-explorer/api/internal/router"
	"github.com/kashmir-explorer/api/internal/search"
	"github.com/kashmir-explorer/api/internal/social"
	"github.com/kashmir-explorer/api/internal/subscription"
	syncpkg "github.com/kashmir-explorer/api/internal/sync"
	"github.com/kashmir-explorer/api/internal/trek"
	"github.com/kashmir-explorer/api/internal/upload"
	"github.com/kashmir-explorer/api/internal/user"
	"github.com/kashmir-explorer/api/internal/wallet"
	"github.com/kashmir-explorer/api/internal/weather"
	"github.com/kashmir-explorer/api/internal/ws"
	"github.com/kashmir-explorer/api/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		// Logger isn't configured yet (it needs cfg), so write to stderr
		// and exit non-zero rather than panicking with a stack trace.
		fmt.Fprintln(os.Stderr, "fatal: config:", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel, cfg.Env)
	slog.SetDefault(log)
	log.Info("kashmir explorer api · booting",
		slog.String("env", cfg.Env), slog.String("port", cfg.Port))

	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      cfg.Env,
			TracesSampleRate: 0.1,
			AttachStacktrace: true,
		}); err != nil {
			log.Error("sentry init", slog.Any("err", err))
		} else {
			defer sentry.Flush(2 * time.Second)
			log.Info("sentry initialised")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Error("db config parse", slog.Any("err", err))
		os.Exit(1)
	}
	// Tuned pool sizing for production load. pgx defaults to MaxConns =
	// max(4, numCPU) and no idle/lifetime caps, which under-utilises larger
	// instances and lets connections go stale behind PgBouncer/Neon.
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Error("db connect", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Error("db ping", slog.Any("err", err))
		os.Exit(1)
	}
	log.Info("db connected")

	// ── Hub + rooms for live broadcasts
	hub := ws.NewHub()
	rooms := ws.NewRooms()

	// Start the external advisory fetcher (NDMA + IMD) in the background.
	// Uses a background context so it outlives the 10s DB-connect timeout.
	fetcherCtx, cancelFetcher := context.WithCancel(context.Background())
	defer cancelFetcher()
	go advisory.NewFetcher(pool, hub).Start(fetcherCtx)

	// ── Domain handlers, assembled into router.Deps.
	deps := router.Deps{
		Cfg:   cfg,
		Log:   log,
		Pool:  pool,
		Hub:   hub,
		Rooms: rooms,

		Dest: destination.New(pool),
		User: user.New(pool),

		Auth:         auth.NewService(pool, cfg.JWT, cfg.OTP, cfg.OAuth),
		Trek:         trek.NewService(pool),
		TrekV3:       trek.NewV3(pool),
		Advisory:     advisory.NewService(pool, hub),
		Weather:      weather.NewService(pool, cfg.OpenWeatherKey),
		Provider:     provider.NewService(pool),
		Booking:      booking.NewService(pool, cfg.Razorpay),
		AI:           ai.NewService(cfg.AnthropicKey, cfg.AnthropicModel, pool),
		Cultural:     cultural.NewService(pool),
		Photo:        photo.NewService(pool),
		Permit:       permit.NewService(pool),
		Upload:       upload.NewService(cfg.R2),
		Sync:         syncpkg.NewService(pool),
		Search:       search.NewService(pool, cfg.VoyageKey),
		Crowd:        crowd.NewService(pool, rooms),
		Groups:       groups.NewService(pool),
		Image:        image.NewService(pool),
		Report:       report.NewService(pool),
		Review:       review.NewService(pool),
		Social:       social.NewService(pool),
		Wallet:       wallet.NewService(pool, cfg.ApplePassTypeID, cfg.OAuth.AppleTeamID),
		Subscription: subscription.NewService(pool, cfg.Razorpay),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.New(deps),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		// WriteTimeout is intentionally 0: /ai/ask streams SSE responses and
		// the WS upgrade endpoints are long-lived, so a server-wide write
		// deadline would sever them. The chi Timeout middleware (15s) bounds
		// the ordinary request handlers instead.
	}

	// Background jobs.
	jobs.Start(pool, hub, log)

	go func() {
		log.Info("listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", slog.Any("err", err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
}
