package router

import (
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

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
	"github.com/kashmir-explorer/api/internal/permit"
	"github.com/kashmir-explorer/api/internal/photo"
	"github.com/kashmir-explorer/api/internal/provider"
	"github.com/kashmir-explorer/api/internal/report"
	"github.com/kashmir-explorer/api/internal/search"
	"github.com/kashmir-explorer/api/internal/subscription"
	syncpkg "github.com/kashmir-explorer/api/internal/sync"
	"github.com/kashmir-explorer/api/internal/trek"
	"github.com/kashmir-explorer/api/internal/upload"
	"github.com/kashmir-explorer/api/internal/user"
	"github.com/kashmir-explorer/api/internal/wallet"
	"github.com/kashmir-explorer/api/internal/weather"
	"github.com/kashmir-explorer/api/internal/ws"
)

var update = flag.Bool("update", false, "regenerate the route golden file")

// testDeps builds a Deps with real (but DB-less) handlers. Route registration
// never touches the pool, so a nil pool is safe; we only need the route tree.
func testDeps() Deps {
	cfg := &config.Config{}
	hub := ws.NewHub()
	rooms := ws.NewRooms()
	return Deps{
		Cfg:          cfg,
		Log:          nil,
		Pool:         nil,
		Hub:          hub,
		Rooms:        rooms,
		Dest:         destination.New(nil),
		User:         user.New(nil),
		Auth:         auth.NewService(nil, cfg.JWT, cfg.OTP, cfg.OAuth),
		Trek:         trek.NewService(nil),
		TrekV3:       trek.NewV3(nil),
		Advisory:     advisory.NewService(nil, hub),
		Weather:      weather.NewService(nil, ""),
		Provider:     provider.NewService(nil),
		Booking:      booking.NewService(nil, cfg.Razorpay),
		AI:           ai.NewService("", "", nil),
		Cultural:     cultural.NewService(nil),
		Photo:        photo.NewService(nil),
		Permit:       permit.NewService(nil),
		Upload:       upload.NewService(cfg.R2),
		Sync:         syncpkg.NewService(nil),
		Search:       search.NewService(nil, ""),
		Crowd:        crowd.NewService(nil, rooms),
		Groups:       groups.NewService(nil),
		Image:        image.NewService(nil),
		Report:       report.NewService(nil),
		Wallet:       wallet.NewService(nil, "", ""),
		Subscription: subscription.NewService(nil, cfg.Razorpay),
	}
}

// walkRoutes returns a sorted "METHOD\tPATH" list of every registered route.
func walkRoutes(t *testing.T) []string {
	t.Helper()
	h := New(testDeps())
	routes, ok := h.(chi.Routes)
	if !ok {
		t.Fatalf("router.New did not return a chi.Routes (got %T)", h)
	}
	var lines []string
	err := chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		lines = append(lines, method+"\t"+route)
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	sort.Strings(lines)
	return lines
}

// TestRouteSnapshot locks the full route table (method + path across all three
// middleware scopes) against a golden file. This is the safety net for the
// layering refactor: any accidental route addition, removal, or rename fails
// here. Regenerate intentionally with `go test ./internal/router -run RouteSnapshot -update`.
func TestRouteSnapshot(t *testing.T) {
	got := strings.Join(walkRoutes(t), "\n") + "\n"
	golden := filepath.Join("testdata", "routes.golden")

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", golden)
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if got != string(want) {
		t.Errorf("route table drifted from golden.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
