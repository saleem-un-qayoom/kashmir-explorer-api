// Package main — Kashmir Explorer API entry point.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sentry "github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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
	"github.com/kashmir-explorer/api/internal/jobs"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/internal/permit"
	"github.com/kashmir-explorer/api/internal/photo"
	"github.com/kashmir-explorer/api/internal/provider"
	"github.com/kashmir-explorer/api/internal/report"
	"github.com/kashmir-explorer/api/internal/search"
	"github.com/kashmir-explorer/api/internal/subscription"
	"github.com/kashmir-explorer/api/internal/sync"
	"github.com/kashmir-explorer/api/internal/trek"
	"github.com/kashmir-explorer/api/internal/upload"
	"github.com/kashmir-explorer/api/internal/user"
	"github.com/kashmir-explorer/api/internal/wallet"
	"github.com/kashmir-explorer/api/internal/weather"
	"github.com/kashmir-explorer/api/internal/ws"
	"github.com/kashmir-explorer/api/pkg/logger"
	"github.com/kashmir-explorer/api/pkg/response"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil { panic(err) }

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
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil { log.Error("db connect", slog.Any("err", err)); os.Exit(1) }
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil { log.Error("db ping", slog.Any("err", err)); os.Exit(1) }
	log.Info("db connected")

	// ── Hub + rooms for live broadcasts
	hub   := ws.NewHub()
	rooms := ws.NewRooms()

	// ── Services
	authSvc   := auth.NewService(pool, cfg.JWT, cfg.OTP, cfg.OAuth)
	destSvc   := destination.NewService(pool)
	trekSvc   := trek.NewService(pool)
	advSvc    := advisory.NewService(pool, hub)
	wthSvc    := weather.NewService(pool, cfg.OpenWeatherKey)
	provSvc   := provider.NewService(pool)
	bookSvc   := booking.NewService(pool, cfg.Razorpay)
	aiSvc     := ai.NewService(cfg.AnthropicKey, cfg.AnthropicModel, pool)
	userSvc   := user.NewService(pool)
	culSvc    := cultural.NewService(pool)
	photoSvc  := photo.NewService(pool)
	permSvc   := permit.NewService(pool)
	upSvc     := upload.NewService(cfg.R2)
	synSvc    := sync.NewService(pool)
	searchSvc := search.NewService(pool, cfg.VoyageKey)
	crowdSvc  := crowd.NewService(pool, rooms)
	groupSvc  := groups.NewService(pool)
	reportSvc := report.NewService(pool)
	walletSvc := wallet.NewService(pool, cfg.ApplePassTypeID, cfg.OAuth.AppleTeamID)
	subSvc    := subscription.NewService(pool, cfg.Razorpay)

	// ── Router
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, mw.Logger(log), mw.Recoverer(log),
		middleware.Timeout(15*time.Second), middleware.Compress(5))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, map[string]any{
			"status": "ok", "service": "kashmir-explorer-api",
			"version": "0.1.0", "time": time.Now().UTC(),
			"ws_clients": hub.Count(),
		})
	})

	// ── WebSocket
	r.Get("/ws/advisories",        hub.HandleWS)
	r.Get("/ws/group/{code}",      rooms.HandleGroup)
	r.Get("/ws/crowd/{slug}",      rooms.HandleCrowd)

	r.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/phone/start",  authSvc.PhoneStart)
			r.Post("/phone/verify", authSvc.PhoneVerify)
			r.Post("/google",       authSvc.Google)
			r.Post("/apple",        authSvc.Apple)
			r.Post("/refresh",      authSvc.Refresh)
		})

		r.Route("/destinations", func(r chi.Router) {
			r.Get("/",         destSvc.List)
			r.Get("/featured", destSvc.Featured)
			r.Get("/trending", destSvc.Trending)
			r.Get("/nearby",   destSvc.Nearby)
			r.Get("/map",      destSvc.Bbox)
			r.Get("/{slug}",   destSvc.Get)
			r.Get("/{slug}/photo-spots", photoSvc.ForDestination)
		})

		r.Route("/treks", func(r chi.Router) {
			r.Get("/",            trekSvc.List)
			r.Get("/{slug}",      trekSvc.Get)
			r.Get("/{slug}/path", trekSvc.Path)
			r.Get("/{slug}/density", crowdSvc.Density)
		})

		// Semantic search (public).
		r.Get("/search", searchSvc.Search)

		r.Get("/categories", destSvc.Categories)
		r.Get("/regions",    destSvc.Regions)

		r.Route("/advisories", func(r chi.Router) {
			r.Get("/",                 advSvc.List)
			r.Get("/destination/{id}", advSvc.ForDestination)
		})
		r.Get("/roads/status", advSvc.RoadStatus)

		r.Get("/weather/destination/{slug}", wthSvc.ForDestination)

		r.Route("/providers", func(r chi.Router) {
			r.Get("/",     provSvc.List)
			r.Get("/{id}", provSvc.Get)
		})

		r.Route("/cultural", func(r chi.Router) {
			r.Get("/food",      culSvc.Food)
			r.Get("/festivals", culSvc.Festivals)
			r.Get("/crafts",    culSvc.Crafts)
			r.Get("/etiquette", culSvc.Etiquette)
		})

		r.Route("/permits", func(r chi.Router) {
			r.Get("/",      permSvc.List)
			r.Get("/check", permSvc.Check)
		})

		r.Route("/ai", func(r chi.Router) {
			r.Post("/plan-trip",      aiSvc.PlanTrip)
			r.Post("/ask",            aiSvc.Ask)            // streaming SSE
			r.Post("/identify-place", aiSvc.IdentifyPlace)  // photo → destination
		})

		r.Post("/webhooks/razorpay", bookSvc.RazorpayWebhook)

		// Authenticated.
		r.Group(func(r chi.Router) {
			r.Use(mw.Auth(cfg.JWT))

			r.Get("/me",   userSvc.Me)
			r.Patch("/me", userSvc.Update)

			r.Route("/saved", func(r chi.Router) {
				r.Get("/",                 userSvc.ListSaved)
				r.Post("/{destinationId}", userSvc.Save)
				r.Delete("/{destinationId}", userSvc.Unsave)
			})

			r.Route("/itineraries", func(r chi.Router) {
				r.Get("/",     userSvc.ListItineraries)
				r.Post("/",    userSvc.CreateItinerary)
				r.Put("/{id}", userSvc.UpdateItinerary)
				r.Delete("/{id}", userSvc.DeleteItinerary)
			})

			r.Route("/bookings", func(r chi.Router) {
				r.Get("/",         bookSvc.List)
				r.Post("/",        bookSvc.Create)
				r.Get("/{id}",     bookSvc.Get)
				r.Post("/{id}/cancel", bookSvc.Cancel)
			})

			r.Post("/sync", synSvc.Apply)
			r.Post("/upload/presign", upSvc.Presign)

			// Trek nav extras (auth required for accountability).
			r.Post("/treks/{slug}/ping",   crowdSvc.Ping)
			r.Post("/treks/{slug}/report", reportSvc.Create)

			// Trip groups (live location share).
			r.Post("/groups",                groupSvc.Create)
			r.Post("/groups/join",           groupSvc.Join)
			r.Get("/groups/{code}",          groupSvc.Get)
			r.Delete("/groups/{code}/leave", groupSvc.Leave)

			// Wallet pass.
			r.Get("/bookings/{id}/wallet", walletSvc.For)

			// Premium subscriptions.
			r.Post("/me/subscribe",    subSvc.Subscribe)
			r.Post("/me/cancel-sub",   subSvc.Cancel)
			r.Get("/me/subscription",  subSvc.Get)
		})

		// Admin (role=admin).
		r.Route("/admin", func(r chi.Router) {
			r.Use(mw.Auth(cfg.JWT), mw.RequireAdmin)

			r.Get("/destinations",             destSvc.AdminList)
			r.Get("/destinations/{id}",        destSvc.AdminGet)
			r.Post("/destinations",            destSvc.AdminCreate)
			r.Put("/destinations/{id}",        destSvc.AdminUpdate)
			r.Delete("/destinations/{id}",     destSvc.AdminDelete)

			r.Post("/treks",                   trekSvc.AdminCreate)
			r.Put("/treks/{id}",               trekSvc.AdminUpdate)

			r.Post("/advisories",              advSvc.AdminCreate)
			r.Put("/advisories/{id}",          advSvc.AdminUpdate)
			r.Delete("/advisories/{id}",       advSvc.AdminDelete)
			r.Put("/roads/{id}/status",        advSvc.AdminUpdateRoad)

			r.Post("/providers/{id}/verify",   provSvc.AdminVerify)
			r.Post("/cultural",                culSvc.AdminCreate)

			// Trek reports queue.
			r.Get("/reports",                  reportSvc.AdminList)
			r.Post("/reports/{id}/resolve",    reportSvc.AdminResolve)

			// Embeddings reindex.
			r.Post("/reindex",                 searchSvc.Reindex)
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
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
