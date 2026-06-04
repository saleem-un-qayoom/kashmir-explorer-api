// Package router builds the HTTP route tree for the Kashmir Explorer API.
//
// It is the single place that knows the cross-cutting wiring — middleware stack,
// CORS, health/readiness, WebSocket upgrades, Swagger UI, the /v1 prefix, and the
// public / authenticated / admin middleware scopes. Domain route knowledge lives
// next to each domain: refactored packages expose scoped registrars
// (e.g. Handler.PublicRoutes / AuthedRoutes / AdminRoutes) which this builder
// mounts inside the appropriate scope. Packages not yet migrated to the layered
// layout are still registered inline here and will move to their own routes.go
// during the rollout.
package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"

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
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/internal/permit"
	"github.com/kashmir-explorer/api/internal/photo"
	"github.com/kashmir-explorer/api/internal/provider"
	"github.com/kashmir-explorer/api/internal/report"
	"github.com/kashmir-explorer/api/internal/review"
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
	"github.com/kashmir-explorer/api/pkg/response"
)

// Deps carries everything the route tree needs: cross-cutting infra plus one
// handle per domain. main.go constructs these and hands them over; the router
// owns no construction itself.
type Deps struct {
	Cfg   *config.Config
	Log   *slog.Logger
	Pool  *pgxpool.Pool
	Hub   *ws.Hub
	Rooms *ws.Rooms

	// Refactored (layered) packages expose scoped registrars.
	Dest *destination.Handler
	User *user.Handler

	// Packages pending migration still expose their Service directly.
	Auth         *auth.Service
	Trek         *trek.Service
	TrekV3       *trek.V3
	Advisory     *advisory.Service
	Weather      *weather.Service
	Provider     *provider.Service
	Booking      *booking.Service
	AI           *ai.Service
	Cultural     *cultural.Service
	Photo        *photo.Service
	Permit       *permit.Service
	Upload       *upload.Service
	Sync         *syncpkg.Service
	Search       *search.Service
	Crowd        *crowd.Service
	Groups       *groups.Service
	Image        *image.Service
	Report       *report.Service
	Review       *review.Service
	Social       *social.Service
	Wallet       *wallet.Service
	Subscription *subscription.Service
}

// New assembles and returns the fully-wired HTTP handler.
func New(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, mw.Logger(d.Log), mw.Recoverer(d.Log),
		middleware.Timeout(15*time.Second), middleware.Compress(5))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   d.Cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	registerHealth(r, d)
	registerWebsockets(r, d)
	registerDocs(r)

	r.Route("/v1", func(r chi.Router) {
		// Per-IP backstop across the whole API surface (DoS / runaway clients).
		// In-memory, so the limit is per-machine; a distributed limiter (Redis)
		// would be the next step once REDIS_URL is wired.
		r.Use(httprate.LimitByIP(300, time.Minute))
		registerPublic(r, d)
		registerAuthed(r, d)
		registerAdmin(r, d)
	})

	return r
}

// ─── Health / readiness ─────────────────────────────────────────

func registerHealth(r chi.Router, d Deps) {
	// Liveness: process is up. Cheap, never touches dependencies.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		response.OK(w, map[string]any{
			"status": "ok", "service": "kashmir-explorer-api",
			"version": "0.1.0", "time": time.Now().UTC(),
			"ws_clients": d.Hub.Count(),
		})
	})

	// Readiness: only "ready" when the database is reachable. Load balancers and
	// orchestrators should gate traffic on this, not /healthz.
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		pingCtx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := d.Pool.Ping(pingCtx); err != nil {
			response.Error(w, http.StatusServiceUnavailable, "not_ready", "database unreachable")
			return
		}
		response.OK(w, map[string]any{"status": "ready"})
	})
}

func registerWebsockets(r chi.Router, d Deps) {
	r.Get("/ws/advisories", d.Hub.HandleWS)
	r.Get("/ws/group/{code}", d.Rooms.HandleGroup)
	r.Get("/ws/crowd/{slug}", d.Rooms.HandleCrowd)
}

// ─── /v1 public ─────────────────────────────────────────────────

func registerPublic(r chi.Router, d Deps) {
	r.Route("/auth", func(r chi.Router) {
		// Auth is the prime abuse target: credential stuffing + (via phone/start)
		// real SMS cost. Group limit guards the whole flow; phone/start is tighter
		// still since each call can send a paid SMS.
		r.Use(httprate.LimitByIP(30, time.Minute))
		r.With(httprate.LimitByIP(5, time.Minute)).Post("/phone/start", d.Auth.PhoneStart)
		r.Post("/phone/verify", d.Auth.PhoneVerify)
		r.Post("/google", d.Auth.Google)
		r.Post("/apple", d.Auth.Apple)
		r.Post("/refresh", d.Auth.Refresh)
	})

	// Destinations + taxonomy (refactored): owns its whole subtree. The
	// photo-spots route lives inside the /destinations group but is served by
	// the photo package, so the handler is injected.
	d.Dest.PublicRoutes(r, d.Photo.ForDestination)
	r.Get("/destinations/{slug}/reviews", d.Review.ListForDestination)

	r.Route("/treks", func(r chi.Router) {
		r.Get("/", d.Trek.List)
		r.Get("/{slug}", d.Trek.Get)
		r.Get("/{slug}/path", d.Trek.Path)
		r.Get("/{slug}/density", d.Crowd.Density)
		r.Get("/{slug}/reports", d.Report.PublicList) // V3 · community trail conditions
		r.Get("/{slug}/reviews", d.Review.ListForTrek)
	})

	// V3 · public share-link viewer (no auth)
	r.Get("/tracks/share/{token}", d.TrekV3.ShareTrack)

	// Semantic search (public) — hits pgvector + embeddings, so rate-limit it.
	r.With(httprate.LimitByIP(60, time.Minute)).Get("/search", d.Search.Search)

	// Public profiles + follower lists (social graph).
	r.Route("/users", func(r chi.Router) {
		r.Get("/{id}", d.Social.Profile)
		r.Get("/{id}/followers", d.Social.Followers)
		r.Get("/{id}/following", d.Social.Following)
	})

	r.Route("/advisories", func(r chi.Router) {
		r.Get("/", d.Advisory.List)
		r.Get("/destination/{id}", d.Advisory.ForDestination)
	})
	r.Get("/roads/status", d.Advisory.RoadStatus)
	r.Get("/roads/status/{id}", d.Advisory.AdminRoadGet)

	r.Get("/weather/destination/{slug}", d.Weather.ForDestination)

	r.Route("/providers", func(r chi.Router) {
		r.Get("/", d.Provider.List)
		r.Get("/{id}", d.Provider.Get)
	})

	r.Route("/cultural", func(r chi.Router) {
		r.Get("/food", d.Cultural.Food)
		r.Get("/food/{id}", d.Cultural.AdminGet)
		r.Get("/festivals", d.Cultural.Festivals)
		r.Get("/festivals/{id}", d.Cultural.AdminGet)
		r.Get("/crafts", d.Cultural.Crafts)
		r.Get("/crafts/{id}", d.Cultural.AdminGet)
		r.Get("/etiquette", d.Cultural.Etiquette)
		r.Get("/etiquette/{id}", d.Cultural.AdminGet)
	})

	r.Route("/permits", func(r chi.Router) {
		r.Get("/", d.Permit.List)
		r.Get("/check", d.Permit.Check)
		r.Get("/{id}", d.Permit.AdminGet)
	})

	r.Route("/images", func(r chi.Router) {
		r.Get("/destination/{id}", d.Image.ForDestination)
		r.Get("/trek/{id}", d.Image.ForTrek)
	})

	r.Route("/ai", func(r chi.Router) {
		// LLM calls are the most expensive endpoints we have — keep them tight.
		r.Use(httprate.LimitByIP(20, time.Minute))
		r.Post("/plan-trip", d.AI.PlanTrip)
		r.Post("/ask", d.AI.Ask)                      // streaming SSE
		r.Post("/identify-place", d.AI.IdentifyPlace) // photo → destination
	})

	r.Post("/webhooks/razorpay", d.Booking.RazorpayWebhook)
}

// ─── /v1 authenticated ──────────────────────────────────────────

func registerAuthed(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(d.Cfg.JWT))

		// User profile / saved / itineraries (refactored).
		d.User.AuthedRoutes(r)

		r.Route("/bookings", func(r chi.Router) {
			r.Get("/", d.Booking.List)
			r.Post("/", d.Booking.Create)
			r.Get("/{id}", d.Booking.Get)
			r.Post("/{id}/cancel", d.Booking.Cancel)
		})

		r.Post("/sync", d.Sync.Apply)
		r.Post("/upload/presign", d.Upload.Presign)

		// Trek nav extras (auth required for accountability).
		r.Post("/treks/{slug}/ping", d.Crowd.Ping)
		r.Post("/treks/{slug}/report", d.Report.Create)

		// Reviews (create/update is an upsert per user+target).
		r.Post("/destinations/{slug}/reviews", d.Review.CreateForDestination)
		r.Post("/treks/{slug}/reviews", d.Review.CreateForTrek)
		r.Get("/me/reviews", d.Review.Mine)
		r.Delete("/reviews/{id}", d.Review.Delete)

		// Social graph: follow/unfollow + activity feed.
		r.Post("/users/{id}/follow", d.Social.Follow)
		r.Delete("/users/{id}/follow", d.Social.Unfollow)
		r.Get("/me/feed", d.Social.Feed)

		// V3 · tracks + summit log
		r.Post("/tracks", d.TrekV3.CreateTrack)
		r.Get("/me/tracks", d.TrekV3.MyTracks)
		r.Post("/treks/{slug}/bag", d.TrekV3.Bag)
		r.Get("/me/completions", d.TrekV3.MyCompletions)

		// Trip groups (live location share).
		r.Post("/groups", d.Groups.Create)
		r.Post("/groups/join", d.Groups.Join)
		r.Get("/groups/{code}", d.Groups.Get)
		r.Delete("/groups/{code}/leave", d.Groups.Leave)

		// Wallet pass.
		r.Get("/bookings/{id}/wallet", d.Wallet.For)

		// Premium subscriptions.
		r.Post("/me/subscribe", d.Subscription.Subscribe)
		r.Post("/me/cancel-sub", d.Subscription.Cancel)
		r.Get("/me/subscription", d.Subscription.Get)
	})
}

// ─── /v1 admin ──────────────────────────────────────────────────

func registerAdmin(r chi.Router, d Deps) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(mw.Auth(d.Cfg.JWT), mw.RequireAdmin)

		// Destinations + categories + regions (refactored).
		d.Dest.AdminRoutes(r)

		// Treks
		r.Get("/treks", d.Trek.AdminList)
		r.Get("/treks/{id}", d.Trek.AdminGet)
		r.Post("/treks", d.Trek.AdminCreate)
		r.Put("/treks/{id}", d.Trek.AdminUpdate)
		r.Delete("/treks/{id}", d.Trek.AdminDelete)

		// Advisories
		r.Post("/advisories", d.Advisory.AdminCreate)
		r.Put("/advisories/{id}", d.Advisory.AdminUpdate)
		r.Delete("/advisories/{id}", d.Advisory.AdminDelete)

		// Roads (status/ prefix matches frontend crud)
		r.Get("/roads/status", d.Advisory.RoadStatus)
		r.Get("/roads/status/{id}", d.Advisory.AdminRoadGet)
		r.Post("/roads/status", d.Advisory.AdminRoadCreate)
		r.Put("/roads/status/{id}", d.Advisory.AdminRoadUpdate)
		r.Delete("/roads/status/{id}", d.Advisory.AdminRoadDelete)
		r.Put("/roads/{id}/status", d.Advisory.AdminUpdateRoad) // legacy compat

		// Permits
		r.Get("/permits/{id}", d.Permit.AdminGet)
		r.Post("/permits", d.Permit.AdminCreate)
		r.Put("/permits/{id}", d.Permit.AdminUpdate)
		r.Delete("/permits/{id}", d.Permit.AdminDelete)

		// Cultural (per-subtype routes)
		r.Route("/cultural", func(r chi.Router) {
			r.Post("/food", d.Cultural.AdminCreateFor("dish"))
			r.Put("/food/{id}", d.Cultural.AdminUpdate)
			r.Delete("/food/{id}", d.Cultural.AdminDelete)

			r.Post("/festivals", d.Cultural.AdminCreateFor("festival"))
			r.Put("/festivals/{id}", d.Cultural.AdminUpdate)
			r.Delete("/festivals/{id}", d.Cultural.AdminDelete)

			r.Post("/crafts", d.Cultural.AdminCreateFor("craft"))
			r.Put("/crafts/{id}", d.Cultural.AdminUpdate)
			r.Delete("/crafts/{id}", d.Cultural.AdminDelete)

			r.Post("/etiquette", d.Cultural.AdminCreateFor("etiquette"))
			r.Put("/etiquette/{id}", d.Cultural.AdminUpdate)
			r.Delete("/etiquette/{id}", d.Cultural.AdminDelete)
		})

		// Photo spots
		r.Get("/photo-spots", d.Photo.AdminList)
		r.Get("/photo-spots/{id}", d.Photo.AdminGet)
		r.Post("/photo-spots", d.Photo.AdminCreate)
		r.Put("/photo-spots/{id}", d.Photo.AdminUpdate)
		r.Delete("/photo-spots/{id}", d.Photo.AdminDelete)

		// Images
		r.Post("/images", d.Image.AdminCreate)
		r.Put("/images/{id}", d.Image.AdminUpdate)
		r.Delete("/images/{id}", d.Image.AdminDelete)

		// Providers
		r.Post("/providers", d.Provider.AdminCreate)
		r.Put("/providers/{id}", d.Provider.AdminUpdate)
		r.Delete("/providers/{id}", d.Provider.AdminDelete)
		r.Post("/providers/{id}/verify", d.Provider.AdminVerify)

		// Legacy: POST /admin/cultural (generic type body)
		r.Post("/cultural", d.Cultural.AdminCreate)

		// Trek reports queue.
		r.Get("/reports", d.Report.AdminList)
		r.Post("/reports/{id}/resolve", d.Report.AdminResolve)

		// Reviews moderation.
		r.Get("/reviews", d.Review.AdminList)
		r.Delete("/reviews/{id}", d.Review.AdminDelete)

		// V3 · all track recordings (moderation / abuse triage)
		r.Get("/tracks", d.TrekV3.AdminTracks)

		// Embeddings reindex.
		r.Post("/reindex", d.Search.Reindex)
	})
}
