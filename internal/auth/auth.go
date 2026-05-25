// Package auth — Phone OTP + Google + Apple sign-in.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/internal/clients"
	"github.com/kashmir-explorer/api/internal/config"
	pkgjwt "github.com/kashmir-explorer/api/pkg/jwt"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool   *pgxpool.Pool
	issuer *pkgjwt.Issuer
	otp    config.OTPConfig
	sms    *clients.MSG91
	google *clients.GoogleVerifier
	apple  *clients.AppleVerifier
}

func NewService(pool *pgxpool.Pool, jwt config.JWTConfig, otp config.OTPConfig, oauth config.OAuthConfig) *Service {
	return &Service{
		pool:   pool,
		issuer: pkgjwt.NewIssuer(jwt.Secret, jwt.RefreshSecret, jwt.AccessTTLHrs, jwt.RefreshTTLDays),
		otp:    otp,
		sms:    clients.NewMSG91(otp.MSG91AuthKey, otp.MSG91TemplateID),
		google: clients.NewGoogleVerifier(oauth.GoogleClientID),
		apple:  clients.NewAppleVerifier(oauth.AppleClientID),
	}
}

// ─── Phone OTP ────────────────────────────────────────────────────

type phoneStartReq struct{ Phone string `json:"phone"` }

func (s *Service) PhoneStart(w http.ResponseWriter, r *http.Request) {
	var body phoneStartReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Phone == "" {
		response.BadRequest(w, "phone is required")
		return
	}
	phone := normalisePhone(body.Phone)
	code := generateOTP()
	hash := hashOTP(code)
	expires := time.Now().Add(10 * time.Minute)

	if _, err := s.pool.Exec(r.Context(), `
		INSERT INTO otp_codes (phone, code_hash, expires_at, attempts)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (phone) DO UPDATE SET code_hash = $2, expires_at = $3, attempts = 0
	`, phone, hash, expires); err != nil {
		response.Internal(w, err)
		return
	}

	if s.otp.DevBypass {
		slog.Info("dev OTP", slog.String("phone", phone), slog.String("code", code))
	} else {
		if err := s.sms.SendOTP(r.Context(), phone, code); err != nil {
			slog.Error("msg91 send failed", slog.Any("err", err))
			response.Internal(w, errors.New("could not deliver OTP"))
			return
		}
	}

	response.OK(w, map[string]any{"sent": true, "expires_in": 600})
}

type phoneVerifyReq struct{ Phone, Code string }

func (s *Service) PhoneVerify(w http.ResponseWriter, r *http.Request) {
	var body phoneVerifyReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Phone == "" || body.Code == "" {
		response.BadRequest(w, "phone and code required")
		return
	}
	phone := normalisePhone(body.Phone)

	devOk := s.otp.DevBypass && body.Code == "000000"
	if !devOk {
		var hash string
		var expires time.Time
		var attempts int
		err := s.pool.QueryRow(r.Context(),
			`SELECT code_hash, expires_at, attempts FROM otp_codes WHERE phone=$1`, phone,
		).Scan(&hash, &expires, &attempts)
		if err != nil { response.Unauthorized(w, "no OTP requested"); return }
		if attempts >= 5 { response.Unauthorized(w, "too many attempts"); return }
		if time.Now().After(expires) { response.Unauthorized(w, "OTP expired"); return }
		if hashOTP(body.Code) != hash {
			_, _ = s.pool.Exec(r.Context(), `UPDATE otp_codes SET attempts = attempts + 1 WHERE phone=$1`, phone)
			response.Unauthorized(w, "incorrect code")
			return
		}
	}

	uid, role, err := s.upsertUserByPhone(r.Context(), phone)
	if err != nil { response.Internal(w, err); return }
	_, _ = s.pool.Exec(r.Context(), `DELETE FROM otp_codes WHERE phone=$1`, phone)

	access, refresh, err := s.issuer.Issue(uid, role)
	if err != nil { response.Internal(w, err); return }
	response.OK(w, tokenEnvelope(uid, role, access, refresh))
}

// ─── OAuth ────────────────────────────────────────────────────────

type oauthReq struct{ IDToken string `json:"id_token"` }

func (s *Service) Google(w http.ResponseWriter, r *http.Request) {
	var body oauthReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.IDToken == "" {
		response.BadRequest(w, "id_token required")
		return
	}
	claims, err := s.google.Verify(r.Context(), body.IDToken)
	if err != nil { response.Unauthorized(w, "Google verification failed: "+err.Error()); return }
	s.upsertAndIssue(w, r, claims)
}

func (s *Service) Apple(w http.ResponseWriter, r *http.Request) {
	var body oauthReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.IDToken == "" {
		response.BadRequest(w, "id_token required")
		return
	}
	claims, err := s.apple.Verify(r.Context(), body.IDToken)
	if err != nil { response.Unauthorized(w, "Apple verification failed: "+err.Error()); return }
	s.upsertAndIssue(w, r, claims)
}

func (s *Service) upsertAndIssue(w http.ResponseWriter, r *http.Request, c *clients.OAuthClaims) {
	uid, role, err := s.upsertUserByOAuth(r.Context(), c)
	if err != nil { response.Internal(w, err); return }
	access, refresh, err := s.issuer.Issue(uid, role)
	if err != nil { response.Internal(w, err); return }
	response.OK(w, tokenEnvelope(uid, role, access, refresh))
}

// ─── Refresh ────────────────────────────────────────────────────

type refreshReq struct{ RefreshToken string `json:"refresh_token"` }

func (s *Service) Refresh(w http.ResponseWriter, r *http.Request) {
	var body refreshReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		response.BadRequest(w, "refresh_token required")
		return
	}
	claims, err := s.issuer.Verify(body.RefreshToken)
	if err != nil { response.Unauthorized(w, "invalid refresh token"); return }
	access, refresh, err := s.issuer.Issue(claims.UserID, claims.Role)
	if err != nil { response.Internal(w, err); return }
	response.OK(w, tokenEnvelope(claims.UserID, claims.Role, access, refresh))
}

// ─── Persistence ─────────────────────────────────────────────────

func (s *Service) upsertUserByPhone(ctx context.Context, phone string) (string, string, error) {
	var id, role string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (phone, provider, role)
		VALUES ($1, 'phone', 'user')
		ON CONFLICT (phone) DO UPDATE SET updated_at = now()
		RETURNING id::text, role
	`, phone).Scan(&id, &role)
	return id, role, err
}

func (s *Service) upsertUserByOAuth(ctx context.Context, c *clients.OAuthClaims) (string, string, error) {
	var id, role string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, provider, provider_id, name, avatar_url, role)
		VALUES ($1, $2, $3, $4, $5, 'user')
		ON CONFLICT (email) DO UPDATE
		  SET provider = EXCLUDED.provider,
		      provider_id = EXCLUDED.provider_id,
		      updated_at = now()
		RETURNING id::text, role
	`, c.Email, c.Provider, c.Sub, c.Name, c.Picture).Scan(&id, &role)
	return id, role, err
}

// ─── Helpers ───────────────────────────────────────────────────

func tokenEnvelope(uid, role, access, refresh string) map[string]any {
	return map[string]any{
		"user":          map[string]string{"id": uid, "role": role},
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "Bearer",
	}
}

func normalisePhone(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	if !strings.HasPrefix(s, "+") { s = "+91" + s }
	return s
}

func generateOTP() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	return fmt.Sprintf("%06d", n)
}
func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
