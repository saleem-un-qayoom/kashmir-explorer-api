// Package wallet — Apple Wallet (.pkpass) generation for bookings.
//
// We emit an unsigned pass JSON bundle. Signing requires the Apple Pass Type
// ID certificate from Apple Developer; without it we return pass.json which
// can be signed by a downstream service. Mobile then uses
// AddPassesViewController to install.
package wallet

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool       *pgxpool.Pool
	passTypeID string
	teamID     string
}

func NewService(pool *pgxpool.Pool, passTypeID, teamID string) *Service {
	return &Service{pool: pool, passTypeID: passTypeID, teamID: teamID}
}

// GET /v1/bookings/{id}/wallet — returns pass.json.
func (s *Service) For(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := mw.UserID(r)

	var (
		ref, status, providerName, providerType string
		startDate, endDate                      time.Time
		guests, total                           int
	)
	err := s.pool.QueryRow(r.Context(), `
		SELECT b.ref, b.status, b.start_date, COALESCE(b.end_date, b.start_date),
		       b.guests, b.total_inr, p.name, p.type
		FROM bookings b JOIN providers p ON p.id = b.provider_id
		WHERE b.id = $1 AND b.user_id = $2
	`, id, userID).Scan(&ref, &status, &startDate, &endDate, &guests, &total, &providerName, &providerType)
	if err != nil { response.NotFound(w, "booking not found"); return }

	if status != "confirmed" && status != "completed" {
		response.BadRequest(w, "wallet pass only available after confirmation")
		return
	}

	serial, authToken := s.ensureWalletCreds(r.Context(), id, ref)

	pass := buildPassJSON(passData{
		PassTypeID:   s.passTypeID,
		TeamID:       s.teamID,
		Serial:       serial,
		AuthToken:    authToken,
		BookingRef:   ref,
		ProviderName: providerName,
		ProviderType: providerType,
		StartDate:    startDate,
		EndDate:      endDate,
		Guests:       guests,
		TotalINR:     total,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pass)
}

func (s *Service) ensureWalletCreds(ctx context.Context, bookingID, ref string) (string, string) {
	var serial, token string
	_ = s.pool.QueryRow(ctx,
		`SELECT COALESCE(wallet_serial, ''), COALESCE(wallet_auth_token, '')
		 FROM bookings WHERE id = $1`, bookingID,
	).Scan(&serial, &token)
	if serial != "" { return serial, token }

	serial = ref
	tokBytes := sha1.Sum([]byte(ref + ":" + bookingID))
	token = hex.EncodeToString(tokBytes[:])
	_, _ = s.pool.Exec(ctx,
		`UPDATE bookings SET wallet_serial = $2, wallet_auth_token = $3 WHERE id = $1`,
		bookingID, serial, token)
	return serial, token
}

type passData struct {
	PassTypeID, TeamID, Serial, AuthToken    string
	BookingRef, ProviderName, ProviderType   string
	StartDate, EndDate                       time.Time
	Guests, TotalINR                         int
}

func buildPassJSON(p passData) map[string]any {
	return map[string]any{
		"formatVersion":       1,
		"passTypeIdentifier":  ifEmpty(p.PassTypeID, "pass.app.kashmir.explorer"),
		"teamIdentifier":      ifEmpty(p.TeamID, "ABCDE12345"),
		"serialNumber":        p.Serial,
		"authenticationToken": p.AuthToken,
		"webServiceURL":       "https://api.kashmir.app/v1/wallet",
		"organizationName":    "Kashmir Explorer",
		"description":         fmt.Sprintf("%s booking · %s", p.ProviderType, p.ProviderName),
		"foregroundColor":     "rgb(245, 235, 220)",
		"backgroundColor":     "rgb(42, 82, 102)",
		"labelColor":          "rgb(232, 137, 58)",
		"barcodes": []map[string]any{{
			"format":          "PKBarcodeFormatQR",
			"message":         p.BookingRef,
			"messageEncoding": "iso-8859-1",
			"altText":         p.BookingRef,
		}},
		"relevantDate": p.StartDate.Format(time.RFC3339),
		"eventTicket": map[string]any{
			"primaryFields": []map[string]any{
				{"key": "provider", "label": "PROVIDER", "value": p.ProviderName},
			},
			"secondaryFields": []map[string]any{
				{"key": "start", "label": "CHECK IN",  "value": p.StartDate.Format("02 Jan 2006")},
				{"key": "end",   "label": "CHECK OUT", "value": p.EndDate.Format("02 Jan 2006")},
			},
			"auxiliaryFields": []map[string]any{
				{"key": "guests", "label": "GUESTS", "value": p.Guests},
				{"key": "total",  "label": "TOTAL",  "value": fmt.Sprintf("₹%d", p.TotalINR)},
			},
			"backFields": []map[string]any{
				{"key": "ref",         "label": "Booking ID",   "value": p.BookingRef},
				{"key": "support",     "label": "Support",      "value": "support@kashmir.app"},
				{"key": "cancellation","label": "Cancellation", "value": "Subject to provider's policy. Open the app to manage."},
			},
		},
	}
}

func ifEmpty(s, fallback string) string {
	if s == "" { return fallback }
	return s
}
