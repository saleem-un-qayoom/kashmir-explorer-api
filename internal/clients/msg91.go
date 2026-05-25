// Package clients · MSG91 — SMS OTP delivery for India numbers.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MSG91 struct {
	AuthKey    string
	TemplateID string
	HTTP       *http.Client
}

func NewMSG91(authKey, templateID string) *MSG91 {
	return &MSG91{
		AuthKey:    authKey,
		TemplateID: templateID,
		HTTP:       &http.Client{Timeout: 8 * time.Second},
	}
}

// SendOTP via MSG91 Flow API. Phone must include country code.
//
// Docs: https://docs.msg91.com/sms/send-sms
func (m *MSG91) SendOTP(ctx context.Context, phone, code string) error {
	if m.AuthKey == "" || m.TemplateID == "" {
		return errors.New("MSG91 not configured (set MSG91_AUTH_KEY + MSG91_TEMPLATE_ID)")
	}

	payload := map[string]any{
		"template_id": m.TemplateID,
		"short_url":   "0",
		"recipients": []map[string]string{
			{
				"mobiles": stripPlus(phone),
				"otp":     code,
			},
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://control.msg91.com/api/v5/flow/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authkey", m.AuthKey)

	res, err := m.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("msg91 request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		buf, _ := io.ReadAll(res.Body)
		return fmt.Errorf("msg91 %d: %s", res.StatusCode, string(buf))
	}
	return nil
}

func stripPlus(s string) string {
	if len(s) > 0 && s[0] == '+' {
		return s[1:]
	}
	return s
}
