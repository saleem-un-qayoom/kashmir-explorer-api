// Package booking — bookings + real Razorpay order creation + webhook.
package booking

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/internal/clients"
	"github.com/kashmir-explorer/api/internal/config"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool *pgxpool.Pool
	rp   *clients.Razorpay
}

func NewService(pool *pgxpool.Pool, cfg config.RazorpayConfig) *Service {
	return &Service{
		pool: pool,
		rp:   clients.NewRazorpay(cfg.KeyID, cfg.KeySecret, cfg.WebhookSecret),
	}
}

type BookingInput struct {
	ProviderID string `json:"provider_id"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date,omitempty"`
	Guests     int    `json:"guests"`
	Notes      string `json:"notes,omitempty"`
}

// Booking / response doc-models (OpenAPI/codegen; handlers emit these fields).
type BookingProvider struct {
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Phone *string `json:"phone,omitempty"`
}

type Booking struct {
	ID        string          `json:"id"`
	Ref       string          `json:"ref"`
	StartDate string          `json:"start_date"`
	EndDate   string          `json:"end_date"`
	Guests    int             `json:"guests"`
	BaseINR   int             `json:"base_inr,omitempty"`
	GstINR    int             `json:"gst_inr,omitempty"`
	FeeINR    int             `json:"fee_inr,omitempty"`
	TotalINR  int             `json:"total_inr"`
	Status    string          `json:"status"`
	Notes     *string         `json:"notes,omitempty"`
	Type      string          `json:"type,omitempty"`
	Provider  BookingProvider `json:"provider"`
}

// BookingOrder is the create response: a booking plus its Razorpay order handle.
type BookingOrder struct {
	ID              string         `json:"id"`
	Ref             string         `json:"ref"`
	RazorpayOrderID string         `json:"razorpay_order_id"`
	RazorpayKeyID   string         `json:"razorpay_key_id"`
	TotalINR        int            `json:"total_inr"`
	Breakdown       map[string]int `json:"breakdown"`
}

// Create godoc
// @Summary  Create a booking + Razorpay order
// @Tags     bookings
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body booking.BookingInput true "Booking request"
// @Success  201 {object} response.Envelope{data=booking.BookingOrder}
// @Failure  400 {object} response.Envelope
// @Router   /v1/bookings [post]
func (s *Service) Create(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	var body BookingInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if body.ProviderID == "" || body.StartDate == "" || body.Guests < 1 {
		response.BadRequest(w, "provider_id, start_date, guests required")
		return
	}

	// Look up price.
	var priceINR int
	var unit string
	if err := s.pool.QueryRow(r.Context(),
		`SELECT price_inr, price_unit FROM providers WHERE id = $1`, body.ProviderID,
	).Scan(&priceINR, &unit); err != nil {
		response.BadRequest(w, "provider not found")
		return
	}

	nights := 1
	if body.EndDate != "" {
		start, _ := time.Parse("2006-01-02", body.StartDate)
		end, _ := time.Parse("2006-01-02", body.EndDate)
		if d := int(end.Sub(start).Hours() / 24); d > 0 {
			nights = d
		}
	}

	base := priceINR
	if unit == "per-night" {
		base = priceINR * nights
	}
	gst := base * 18 / 100
	fee := base * 3 / 100
	total := base + gst + fee

	ref := fmt.Sprintf("KEX-%05d", randInt(99999))

	// Create the Razorpay order (amount in paise).
	order, err := s.rp.CreateOrder(r.Context(), clients.OrderRequest{
		AmountPaise:    total * 100,
		Currency:       "INR",
		Receipt:        ref,
		PaymentCapture: true,
		Notes: map[string]string{
			"user_id":     userID,
			"provider_id": body.ProviderID,
		},
	})
	if err != nil {
		response.Internal(w, fmt.Errorf("razorpay order failed: %w", err))
		return
	}

	var id string
	if err := s.pool.QueryRow(r.Context(), `
		INSERT INTO bookings (ref, user_id, provider_id, start_date, end_date, guests,
		                      base_inr, gst_inr, fee_inr, total_inr, status, razorpay_order_id, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', $11, $12)
		RETURNING id::text
	`, ref, userID, body.ProviderID, body.StartDate, body.EndDate, body.Guests,
		base, gst, fee, total, order.ID, body.Notes).Scan(&id); err != nil {
		response.Internal(w, err)
		return
	}

	response.Created(w, map[string]any{
		"id":                id,
		"ref":               ref,
		"razorpay_order_id": order.ID,
		"razorpay_key_id":   s.rp.KeyID, // mobile SDK needs this
		"total_inr":         total,
		"breakdown": map[string]int{
			"base": base, "gst": gst, "fee": fee, "total": total,
		},
	})
}

// List godoc
// @Summary  List the current user's bookings
// @Tags     bookings
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]booking.Booking}
// @Router   /v1/bookings [get]
func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	rows, err := s.pool.Query(r.Context(), `
		SELECT b.id::text, b.ref, b.start_date, COALESCE(b.end_date, b.start_date),
		       b.guests, b.total_inr, b.status, p.name, p.type, p.phone
		FROM bookings b JOIN providers p ON p.id = b.provider_id
		WHERE b.user_id = $1 ORDER BY b.start_date DESC
	`, userID)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, ref, status, name, typ string
		var phone *string
		var start, end time.Time
		var guests, total int
		if err := rows.Scan(&id, &ref, &start, &end, &guests, &total, &status, &name, &typ, &phone); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{
			"id": id, "ref": ref,
			"start_date": start.Format("2006-01-02"),
			"end_date":   end.Format("2006-01-02"),
			"guests":     guests, "total_inr": total, "status": status,
			"provider": map[string]any{"name": name, "type": typ, "phone": phone},
			"type":     typ,
		})
	}
	response.OK(w, out)
}

// Get godoc
// @Summary  Get a booking
// @Tags     bookings
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Booking ID"
// @Success  200 {object} response.Envelope{data=booking.Booking}
// @Failure  404 {object} response.Envelope
// @Router   /v1/bookings/{id} [get]
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := mw.UserID(r)
	var (
		bid, ref, status, name, typ   string
		notes, phone                  *string
		start, end                    time.Time
		guests, base, gst, fee, total int
	)
	err := s.pool.QueryRow(r.Context(), `
		SELECT b.id::text, b.ref, b.start_date, COALESCE(b.end_date, b.start_date),
		       b.guests, b.base_inr, b.gst_inr, b.fee_inr, b.total_inr,
		       b.status, b.notes, p.name, p.type, p.phone
		FROM bookings b JOIN providers p ON p.id = b.provider_id
		WHERE b.id = $1 AND b.user_id = $2
	`, id, userID).Scan(&bid, &ref, &start, &end, &guests, &base, &gst, &fee, &total,
		&status, &notes, &name, &typ, &phone)
	if err != nil {
		response.NotFound(w, "booking not found")
		return
	}
	response.OK(w, map[string]any{
		"id": bid, "ref": ref,
		"start_date": start.Format("2006-01-02"),
		"end_date":   end.Format("2006-01-02"),
		"guests":     guests, "base_inr": base, "gst_inr": gst, "fee_inr": fee,
		"total_inr": total, "status": status, "notes": notes,
		"provider": map[string]any{"name": name, "type": typ, "phone": phone},
	})
}

// Cancel godoc
// @Summary  Cancel a booking
// @Tags     bookings
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Booking ID"
// @Success  200 {object} response.Envelope
// @Router   /v1/bookings/{id}/cancel [post]
func (s *Service) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := mw.UserID(r)
	if _, err := s.pool.Exec(r.Context(), `
		UPDATE bookings SET status='cancelled', updated_at=now()
		WHERE id=$1 AND user_id=$2
	`, id, userID); err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"status": "cancelled"})
}

// RazorpayWebhook godoc
// @Summary  Razorpay payment webhook (HMAC-verified)
// @Tags     webhooks
// @Accept   json
// @Success  200 {string} string "ok"
// @Failure  401 {object} response.Envelope
// @Router   /v1/webhooks/razorpay [post]
func (s *Service) RazorpayWebhook(w http.ResponseWriter, r *http.Request) {
	sig := r.Header.Get("X-Razorpay-Signature")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w, "body read failed")
		return
	}

	if !s.rp.VerifyWebhookSignature(body, sig) {
		response.Unauthorized(w, "invalid signature")
		return
	}

	var payload struct {
		Event   string `json:"event"`
		Payload struct {
			Payment struct {
				Entity struct {
					OrderID string `json:"order_id"`
					ID      string `json:"id"`
					Status  string `json:"status"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}

	switch payload.Event {
	case "payment.captured", "order.paid":
		_, _ = s.pool.Exec(r.Context(), `
			UPDATE bookings SET status='confirmed', razorpay_payment_id=$1, updated_at=now()
			WHERE razorpay_order_id=$2
		`, payload.Payload.Payment.Entity.ID, payload.Payload.Payment.Entity.OrderID)
	case "payment.failed":
		_, _ = s.pool.Exec(r.Context(), `
			UPDATE bookings SET status='cancelled', updated_at=now()
			WHERE razorpay_order_id=$1 AND status='pending'
		`, payload.Payload.Payment.Entity.OrderID)
	}
	w.WriteHeader(http.StatusOK)
}

func randInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}
