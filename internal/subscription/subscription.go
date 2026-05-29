// Package subscription — premium plans via Razorpay Subscriptions.
//
//	POST /v1/me/subscribe    · create Razorpay subscription, return checkout URL
//	POST /v1/me/cancel-sub   · request cancellation (Razorpay-side)
//	GET  /v1/me/subscription · current state
//
// Webhook events are dispatched here from booking.RazorpayWebhook.
package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/internal/clients"
	"github.com/kashmir-explorer/api/internal/config"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool *pgxpool.Pool
	rp   *clients.Razorpay
	cfg  config.RazorpayConfig
}

func NewService(pool *pgxpool.Pool, cfg config.RazorpayConfig) *Service {
	return &Service{
		pool: pool,
		rp:   clients.NewRazorpay(cfg.KeyID, cfg.KeySecret, cfg.WebhookSecret),
		cfg:  cfg,
	}
}

type subscribeReq struct {
	Plan string `json:"plan"` // 'monthly' | 'yearly'
}

// POST /v1/me/subscribe
func (s *Service) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	var body subscribeReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "plan required")
		return
	}

	planID, err := s.razorpayPlanID(body.Plan)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	subID, err := s.rp.CreateSubscription(r.Context(), planID, map[string]string{
		"user_id": userID,
	})
	if err != nil {
		response.Internal(w, err)
		return
	}

	if _, err := s.pool.Exec(r.Context(), `
		INSERT INTO subscriptions (user_id, plan, status, razorpay_subscription_id)
		VALUES ($1, $2, 'pending', $3)
		ON CONFLICT (user_id) DO UPDATE
		  SET plan = $2, status = 'pending', razorpay_subscription_id = $3, cancelled_at = NULL
	`, userID, body.Plan, subID); err != nil {
		response.Internal(w, err)
		return
	}

	response.Created(w, map[string]any{
		"razorpay_subscription_id": subID,
		"razorpay_key_id":          s.cfg.KeyID,
		"plan":                     body.Plan,
		"checkout_url":             fmt.Sprintf("https://checkout.razorpay.com/v1/subscription_button.html?subscription_id=%s", subID),
	})
}

// GET /v1/me/subscription
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	var plan, status string
	var periodEnd *time.Time
	err := s.pool.QueryRow(r.Context(), `
		SELECT plan, status, current_period_end
		FROM subscriptions WHERE user_id = $1
	`, userID).Scan(&plan, &status, &periodEnd)
	if err != nil {
		response.OK(w, map[string]any{"status": "none"})
		return
	}
	response.OK(w, map[string]any{
		"plan":               plan,
		"status":             status,
		"current_period_end": periodEnd,
	})
}

// POST /v1/me/cancel-sub
func (s *Service) Cancel(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	var rpID string
	if err := s.pool.QueryRow(r.Context(),
		`SELECT razorpay_subscription_id FROM subscriptions WHERE user_id = $1`,
		userID).Scan(&rpID); err != nil {
		response.NotFound(w, "no active subscription")
		return
	}
	if err := s.rp.CancelSubscription(r.Context(), rpID, true); err != nil {
		response.Internal(w, err)
		return
	}
	_, _ = s.pool.Exec(r.Context(),
		`UPDATE subscriptions SET cancelled_at = now() WHERE user_id = $1`, userID)
	response.OK(w, map[string]string{"status": "cancellation_scheduled"})
}

func (s *Service) razorpayPlanID(plan string) (string, error) {
	switch plan {
	case "monthly":
		if s.cfg.KeyID == "" {
			return "", errors.New("razorpay not configured")
		}
		return "plan_kashmir_monthly_199_inr", nil
	case "yearly":
		return "plan_kashmir_yearly_999_inr", nil
	default:
		return "", errors.New("plan must be 'monthly' or 'yearly'")
	}
}

/* ─── Webhook hook — called from booking.RazorpayWebhook ──── */

func (s *Service) HandleEvent(ctx context.Context, event string, payload json.RawMessage) {
	type evtBody struct {
		Subscription struct {
			Entity struct {
				ID         string `json:"id"`
				Status     string `json:"status"`
				CurrentEnd int64  `json:"current_end"`
				CustomerID string `json:"customer_id"`
			} `json:"entity"`
		} `json:"subscription"`
	}
	var ev evtBody
	if json.Unmarshal(payload, &ev) != nil {
		return
	}
	sid := ev.Subscription.Entity.ID
	if sid == "" {
		return
	}

	periodEnd := time.Unix(ev.Subscription.Entity.CurrentEnd, 0)
	switch event {
	case "subscription.charged", "subscription.activated":
		_, _ = s.pool.Exec(ctx, `
			UPDATE subscriptions SET status='active', current_period_end=$2,
			       razorpay_customer_id=$3
			WHERE razorpay_subscription_id=$1
		`, sid, periodEnd, ev.Subscription.Entity.CustomerID)
		_, _ = s.pool.Exec(ctx, `
			UPDATE users SET is_premium=TRUE
			WHERE id = (SELECT user_id FROM subscriptions WHERE razorpay_subscription_id=$1)
		`, sid)
	case "subscription.cancelled", "subscription.completed":
		_, _ = s.pool.Exec(ctx, `
			UPDATE subscriptions SET status='cancelled', cancelled_at=now()
			WHERE razorpay_subscription_id=$1
		`, sid)
		_, _ = s.pool.Exec(ctx, `
			UPDATE users SET is_premium=FALSE
			WHERE id = (SELECT user_id FROM subscriptions WHERE razorpay_subscription_id=$1)
		`, sid)
	case "subscription.halted", "subscription.paused":
		_, _ = s.pool.Exec(ctx, `
			UPDATE subscriptions SET status='past_due'
			WHERE razorpay_subscription_id=$1
		`, sid)
	}
}
