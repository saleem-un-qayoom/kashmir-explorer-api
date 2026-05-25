// Package clients · Razorpay — order creation + capture verification.
//
// Docs: https://razorpay.com/docs/api/orders/
package clients

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Razorpay struct {
	KeyID         string
	KeySecret     string
	WebhookSecret string
	HTTP          *http.Client
}

func NewRazorpay(keyID, keySecret, webhookSecret string) *Razorpay {
	return &Razorpay{
		KeyID:         keyID,
		KeySecret:     keySecret,
		WebhookSecret: webhookSecret,
		HTTP:          &http.Client{Timeout: 10 * time.Second},
	}
}

type OrderRequest struct {
	AmountPaise int               `json:"amount"`   // in smallest unit
	Currency    string            `json:"currency"` // 'INR'
	Receipt     string            `json:"receipt"`
	Notes       map[string]string `json:"notes,omitempty"`
	PaymentCapture bool           `json:"payment_capture"` // true = auto-capture
}

type Order struct {
	ID          string `json:"id"`
	Entity      string `json:"entity"`
	Amount      int    `json:"amount"`
	Currency    string `json:"currency"`
	Receipt     string `json:"receipt"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
}

// CreateOrder allocates a Razorpay order which the mobile SDK then opens.
func (r *Razorpay) CreateOrder(ctx context.Context, req OrderRequest) (*Order, error) {
	if r.KeyID == "" || r.KeySecret == "" {
		return nil, errors.New("razorpay not configured")
	}
	body, _ := json.Marshal(req)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.razorpay.com/v1/orders", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(r.KeyID + ":" + r.KeySecret))
	httpReq.Header.Set("Authorization", "Basic "+auth)

	res, err := r.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		buf, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("razorpay %d: %s", res.StatusCode, string(buf))
	}
	var o Order
	if err := json.NewDecoder(res.Body).Decode(&o); err != nil {
		return nil, err
	}
	return &o, nil
}

// VerifyWebhookSignature checks the X-Razorpay-Signature header.
func (r *Razorpay) VerifyWebhookSignature(rawBody []byte, signatureHex string) bool {
	mac := hmac.New(sha256.New, []byte(r.WebhookSecret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signatureHex))
}

// VerifyPaymentSignature — for client-side checkout success (razorpay_signature
// header in the success callback). Hash format:
//   HMAC_SHA256(orderID + "|" + paymentID, key_secret)
func (r *Razorpay) VerifyPaymentSignature(orderID, paymentID, signatureHex string) bool {
	mac := hmac.New(sha256.New, []byte(r.KeySecret))
	mac.Write([]byte(orderID + "|" + paymentID))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signatureHex))
}

/* ─── Subscriptions ───────────────────────────────────────── */

type subscriptionResp struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// CreateSubscription on a pre-existing plan ID. Notes attached for webhook
// reconciliation.
func (r *Razorpay) CreateSubscription(ctx context.Context, planID string, notes map[string]string) (string, error) {
	if r.KeyID == "" || r.KeySecret == "" {
		return "", errors.New("razorpay not configured")
	}
	body, _ := json.Marshal(map[string]any{
		"plan_id":         planID,
		"customer_notify": 1,
		"total_count":     120, // 10 years monthly OR 120 years yearly — effectively forever
		"notes":           notes,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.razorpay.com/v1/subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(r.KeyID+":"+r.KeySecret)))

	res, err := r.HTTP.Do(req)
	if err != nil { return "", err }
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		buf, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("razorpay sub %d: %s", res.StatusCode, string(buf))
	}
	var out subscriptionResp
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil { return "", err }
	return out.ID, nil
}

// CancelSubscription — `cancelAtCycleEnd=true` is friendlier (lets user
// finish what they paid for).
func (r *Razorpay) CancelSubscription(ctx context.Context, subID string, cancelAtCycleEnd bool) error {
	body, _ := json.Marshal(map[string]any{"cancel_at_cycle_end": cancelAtCycleEnd})
	url := fmt.Sprintf("https://api.razorpay.com/v1/subscriptions/%s/cancel", subID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(r.KeyID+":"+r.KeySecret)))

	res, err := r.HTTP.Do(req)
	if err != nil { return err }
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		buf, _ := io.ReadAll(res.Body)
		return fmt.Errorf("razorpay cancel %d: %s", res.StatusCode, string(buf))
	}
	return nil
}
