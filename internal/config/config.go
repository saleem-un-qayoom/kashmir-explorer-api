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
	R2              R2Config
	OpenWeatherKey  string
	AnthropicKey    string
	AnthropicModel  string
	VoyageKey       string // text embeddings for pgvector
	ApplePassTypeID string // Apple Wallet pass type identifier
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

type R2Config struct {
	AccountID       string
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
		R2: R2Config{
			AccountID:       os.Getenv("R2_ACCOUNT_ID"),
			AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
			Bucket:          env("R2_BUCKET", "kashmir-uploads"),
			PublicBase:      env("R2_PUBLIC_BASE", ""),
		},
		OpenWeatherKey:  os.Getenv("OPENWEATHERMAP_API_KEY"),
		AnthropicKey:    os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:  env("ANTHROPIC_MODEL", "claude-sonnet-4-7-20251101"),
		VoyageKey:       os.Getenv("VOYAGE_API_KEY"),
		ApplePassTypeID: env("APPLE_PASS_TYPE_ID", "pass.app.kashmir.explorer"),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cfg.JWT.Secret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	if cfg.Env == "production" && cfg.OTP.DevBypass {
		return nil, fmt.Errorf("OTP_DEV_BYPASS must be false in production")
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
