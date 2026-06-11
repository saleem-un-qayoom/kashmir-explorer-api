// Package config — environment-driven configuration loader.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port            string
	Env             string
	LogLevel        string
	DatabaseURL     string
	RedisURL        string
	AllowedOrigins  []string
	JWT             JWTConfig
	OTP             OTPConfig
	OAuth           OAuthConfig
	Razorpay        RazorpayConfig
	Supabase        SupabaseConfig
	OpenWeatherKey  string
	AnthropicKey    string
	AnthropicModel  string
	VoyageKey       string // text embeddings for pgvector
	ApplePassTypeID string // Apple Wallet pass type identifier
	ContourTilesDir string // pre-built contour vector tiles ({z}/{x}/{y}.pbf)

	// External advisory sources polled by internal/advisory. Set to an empty
	// string to disable that source. Endpoints are not always stable — leave
	// blank in dev if a source is misbehaving, the fetcher will skip it.
	NDMAURL string
	IMDURL  string
}

type JWTConfig struct {
	Secret         string
	RefreshSecret  string
	AccessTTLHrs   int
	RefreshTTLDays int
}

type OTPConfig struct {
	MSG91AuthKey    string
	MSG91TemplateID string
	DevBypass       bool
}

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	AppleTeamID        string
	AppleClientID      string
}

type RazorpayConfig struct {
	KeyID         string
	KeySecret     string
	WebhookSecret string
}

// SupabaseConfig configures Supabase Storage's S3-compatible endpoint.
//
//	ProjectRef — the project ref, i.e. the subdomain of *.supabase.co.
//	Region     — the project's region (e.g. ap-south-1); used in SigV4.
//	S3 access keys are created under Storage → Settings → S3 access keys.
type SupabaseConfig struct {
	ProjectRef      string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBase      string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:           env("PORT", "8080"),
		Env:            env("ENV", "development"),
		LogLevel:       env("LOG_LEVEL", "info"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       env("REDIS_URL", "redis://localhost:6379"),
		AllowedOrigins: splitCSV(env("ALLOWED_ORIGINS", "*")),
		JWT: JWTConfig{
			Secret:         os.Getenv("JWT_SECRET"),
			RefreshSecret:  os.Getenv("JWT_REFRESH_SECRET"),
			AccessTTLHrs:   atoi(env("JWT_ACCESS_TTL_HOURS", "24"), 24),
			RefreshTTLDays: atoi(env("JWT_REFRESH_TTL_DAYS", "30"), 30),
		},
		OTP: OTPConfig{
			MSG91AuthKey:    os.Getenv("MSG91_AUTH_KEY"),
			MSG91TemplateID: os.Getenv("MSG91_TEMPLATE_ID"),
			DevBypass:       env("OTP_DEV_BYPASS", "false") == "true",
		},
		OAuth: OAuthConfig{
			GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			AppleTeamID:        os.Getenv("APPLE_TEAM_ID"),
			AppleClientID:      os.Getenv("APPLE_CLIENT_ID"),
		},
		Razorpay: RazorpayConfig{
			KeyID:         os.Getenv("RAZORPAY_KEY_ID"),
			KeySecret:     os.Getenv("RAZORPAY_KEY_SECRET"),
			WebhookSecret: os.Getenv("RAZORPAY_WEBHOOK_SECRET"),
		},
		Supabase: SupabaseConfig{
			ProjectRef:      os.Getenv("SUPABASE_PROJECT_REF"),
			Region:          env("SUPABASE_REGION", "ap-south-1"),
			AccessKeyID:     os.Getenv("SUPABASE_S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("SUPABASE_S3_SECRET_ACCESS_KEY"),
			Bucket:          env("SUPABASE_BUCKET", "kashmir-uploads"),
			PublicBase:      env("SUPABASE_PUBLIC_BASE", ""),
		},
		OpenWeatherKey:  os.Getenv("OPENWEATHERMAP_API_KEY"),
		AnthropicKey:    os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:  env("ANTHROPIC_MODEL", "claude-sonnet-4-7-20251101"),
		VoyageKey:       os.Getenv("VOYAGE_API_KEY"),
		ApplePassTypeID: env("APPLE_PASS_TYPE_ID", "pass.app.kashmir.explorer"),
		ContourTilesDir: env("CONTOUR_TILES_DIR", "data/contour-tiles"),

		NDMAURL: env("NDMA_URL", "https://sachet.ndma.gov.in/cap_public_website/getAllActiveWarnings"),
		IMDURL:  env("IMD_URL", "https://mausam.imd.gov.in/backend/website/district-level-warning"),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	// The dev OTP bypass (accepts code 000000) is a development-only escape hatch.
	// Honour it *only* when ENV=development so it can never leak into staging or
	// production, even if the flag is set by mistake.
	if cfg.OTP.DevBypass && cfg.Env != "development" {
		return nil, fmt.Errorf("OTP_DEV_BYPASS may only be enabled when ENV=development (got %q)", cfg.Env)
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func atoi(s string, fallback int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return fallback
}
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
